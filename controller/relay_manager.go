package controller

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/heap"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/hypersdk/vm"
	messagenet "github.com/AnomalyFi/nodekit-seq/messagenet"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"go.uber.org/zap"
)

type RelayManager struct {
	vm        *vm.VM
	snowCtx   *snow.Context
	appSender common.AppSender
	l         sync.Mutex
	requestID uint32

	pendingJobs *heap.Heap[*relayJob, int64]
	jobs        map[uint32]*relayJob
	done        chan struct{}

	messagenet *messagenet.MessageNet
}

type relayJob struct {
	nodeIDs   set.Set[ids.NodeID]
	relayerID int
	msg       []byte
}

var maxOutStanding = 1000

func NewRelayManager(vm *vm.VM, messagenet *messagenet.MessageNet, snowCtx *snow.Context) *RelayManager {
	return &RelayManager{
		vm:          vm,
		snowCtx:     snowCtx,
		pendingJobs: heap.New[*relayJob, int64](64, true),
		jobs:        make(map[uint32]*relayJob),
		done:        make(chan struct{}),
		messagenet:  messagenet,
	}
}

func (r *RelayManager) Run(appSender common.AppSender) {
	r.appSender = appSender
	r.vm.Logger().Info("starting relay manager")
	defer close(r.done)

	t := time.NewTicker(5 * time.Millisecond)
	defer t.Stop()
	ctx := context.Background()
	for {
		select {
		case <-t.C:
			r.l.Lock()
			if r.pendingJobs.Len() > 0 && len(r.jobs) < maxOutStanding {
				first := r.pendingJobs.First()
				r.pendingJobs.Pop()
				job := first.Item
				if err := r.request(ctx, job); err != nil {
					r.vm.Logger().Error("error sending relay request", zap.Error(err))
				}
			}
			l := r.pendingJobs.Len()
			r.l.Unlock()
			r.vm.Logger().Debug("checked for ready jobs", zap.Int("pending", l))
		case <-r.vm.StopChan():
			r.vm.Logger().Info("stopping relay manager")
			return
		}
	}
}

// you must hold [r.l] when calling this function
func (r *RelayManager) request(
	ctx context.Context,
	j *relayJob,
) error {
	requestID := r.requestID
	r.requestID++
	r.jobs[requestID] = j
	initial := consts.IntLen + len(j.msg)
	p := codec.NewWriter(initial, initial*2)
	p.PackInt(j.relayerID)
	p.PackBytes(j.msg)
	return r.appSender.SendAppRequest(ctx, j.nodeIDs, requestID, p.Bytes())
}

// incoming requests from other nodes.
func (r *RelayManager) AppRequest(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	request []byte,
) error {
	reader := codec.NewReader(request, consts.MaxInt)
	relayerID := reader.UnpackInt(true)
	var msg []byte
	reader.UnpackBytes(-1, false, &msg)
	// send the request to relayer
	if err := r.messagenet.SendToClient(relayerID, nodeID, msg); err != nil {
		r.vm.Logger().Info("messagenet error: %s", zap.Error(err))
		// don't send back any response, if server did not exist.
		return nil
	}
	return r.appSender.SendAppResponse(ctx, nodeID, requestID, nil)
}

func (r *RelayManager) HandleResponse(requestID uint32, response []byte) error {
	r.l.Lock()
	_, ok := r.jobs[requestID]
	delete(r.jobs, requestID)
	r.l.Unlock()

	if !ok {
		return nil
	}

	r.snowCtx.Log.Info("received response", zap.Uint32("requestID", requestID), zap.Int("responseLen", len(response)))
	return nil
}

func (r *RelayManager) HandleRequestFailed(requestID uint32) error {
	r.l.Lock()
	_, ok := r.jobs[requestID]
	delete(r.jobs, requestID)
	r.l.Unlock()
	if !ok {
		return nil
	}

	r.snowCtx.Log.Info("request failed", zap.Uint32("requestID", requestID))
	return nil
}

func (r *RelayManager) Done() {
	<-r.done
}

func (r *RelayManager) SendRequestToAll(
	ctx context.Context,
	relayerID int,
	data []byte,
) error {
	height, err := r.snowCtx.ValidatorState.GetCurrentHeight(ctx)
	if err != nil {
		r.snowCtx.Log.Error("unable to get current p-chain height", zap.Error(err))
		return fmt.Errorf("unable to get current p-chain height: %w", err)
	}

	validators, err := r.snowCtx.ValidatorState.GetValidatorSet(
		ctx,
		height,
		r.snowCtx.SubnetID,
	)
	if err != nil {
		r.snowCtx.Log.Error("unable to get validator set", zap.Error(err))
		return fmt.Errorf("unable to get validator set: %w", err)
	}

	newSet := set.NewSet[ids.NodeID](len(validators))
	for nodeID := range validators {
		if nodeID == r.snowCtx.NodeID {
			continue
		}
		newSet.Add(nodeID)
	}

	idb := make([]byte, len(data)+8)
	binary.BigEndian.PutUint64(idb, uint64(relayerID))
	copy(idb[8:], data)
	id := utils.ToID(idb)

	r.pendingJobs.Push(&heap.Entry[*relayJob, int64]{
		ID: id,
		Item: &relayJob{
			nodeIDs:   newSet,
			relayerID: relayerID,
			msg:       data,
		},
		Index: r.pendingJobs.Len(),
	})
	return nil
}

func (r *RelayManager) SendRequestToIndividual(
	ctx context.Context,
	relayerID int,
	nodeID ids.NodeID,
	data []byte,
) error {
	newSet := set.NewSet[ids.NodeID](1)
	newSet.Add(nodeID)

	idb := make([]byte, len(data)+8)
	binary.BigEndian.PutUint64(idb, uint64(relayerID))
	copy(idb[8:], data)
	id := utils.ToID(idb)

	r.pendingJobs.Push(&heap.Entry[*relayJob, int64]{
		ID: id,
		Item: &relayJob{
			nodeIDs:   newSet,
			relayerID: relayerID,
			msg:       data,
		},
		Index: r.pendingJobs.Len(),
	})
	return nil
}

// data is marshalled message
func (r *RelayManager) SignAndSendRequestToIndividual(
	ctx context.Context,
	relayerID int,
	nodeID ids.NodeID,
	identificationByte byte,
	data []byte,
) error {
	sigMsgBytes, err := SignRelayManagerMessage(identificationByte, data, r.snowCtx.NetworkID, r.snowCtx.ChainID, r.snowCtx.WarpSigner, bls.PublicKeyToCompressedBytes(r.snowCtx.PublicKey))
	if err != nil {
		r.snowCtx.Log.Error("unable to sign and send request to all", zap.Error(err))
	}
	r.SendRequestToIndividual(ctx, relayerID, nodeID, sigMsgBytes)
	return nil
}

// @todo update relayer to use bls.PublicKeyFromCompressedBytes
func (r *RelayManager) SignAndSendRequestToAll(
	ctx context.Context,
	relayerID int,
	identificationByte byte,
	data []byte,
) error {
	sigMsgBytes, err := SignRelayManagerMessage(identificationByte, data, r.snowCtx.NetworkID, r.snowCtx.ChainID, r.snowCtx.WarpSigner, bls.PublicKeyToCompressedBytes(r.snowCtx.PublicKey))
	if err != nil {
		r.snowCtx.Log.Error("unable to sign and send request to all", zap.Error(err))
	}
	r.SendRequestToAll(ctx, relayerID, sigMsgBytes)
	return nil
}

func SignRelayManagerMessage(
	identificationByte byte,
	data []byte,
	networkID uint32,
	chainID ids.ID,
	signer warp.Signer,
	publicKeyBytes []byte,
) ([]byte, error) {
	// prepare warp message & sign.
	uSigWarpMsg, err := warp.NewUnsignedMessage(networkID, chainID, data)
	if err != nil {
		return nil, fmt.Errorf("unable to create unsigned message: %w", err)
	}

	signature, err := signer.Sign(uSigWarpMsg)
	if err != nil {
		return nil, fmt.Errorf("unable to sign message: %w", err)
	}

	sigMsg := messagenet.SignedMessage{
		PublicKeyBytes:       publicKeyBytes,
		SignatureBytes:       signature,
		UnsignedMessageBytes: data,
	}

	sigMsgBytes, err := json.Marshal(sigMsg)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal signed message: %w", err)
	}
	// append client side identification byte.
	sigMsgBytes = append([]byte{identificationByte}, sigMsgBytes...)
	return sigMsgBytes, nil
}
