package controller

import (
	"context"
	"sync"
	"time"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/heap"
	"github.com/AnomalyFi/hypersdk/vm"
	serverless "github.com/AnomalyFi/nodekit-seq/server-less"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"go.uber.org/zap"
)

type RelayManager struct {
	vm        *vm.VM
	appSender common.AppSender
	l         sync.Mutex
	requestID uint32

	pendingJobs *heap.Heap[*relayJob, int64]
	jobs        map[uint32]*relayJob
	done        chan struct{}

	serverless *serverless.ServerLess
}

// @todo relayer should not be able to send any messages, if it joins in middle of the window.
// where should the above check happen??
type relayJob struct {
	nodeID    ids.NodeID
	relayerID int
	//@todo necessary data provided to other validators/relayers to verify that, the said relayer/validator has successfully submitted the claimed data
	msg []byte // marshalled json
}

func NewRelayManager(vm *vm.VM, serverless *serverless.ServerLess) *RelayManager {
	return &RelayManager{
		vm:          vm,
		pendingJobs: heap.New[*relayJob, int64](64, true),
		jobs:        make(map[uint32]*relayJob),
		done:        make(chan struct{}),
		serverless:  serverless,
	}
}

// A selected relayer informs that he is picking to relay the data
// when data has been successfully relayed, he sends a message to the other validators/relayers along with txID
// other relayers/validators verify the validity of submission and then sign a signature stating validator has successfully completed the job.
// then the relayer aggregates the signatures, and sends to SEQ for his relay rewards?

func (r *RelayManager) Run(appSender common.AppSender) {
	r.appSender = appSender
	r.vm.Logger().Info("starting relay manager")
	defer close(r.done)
	//@todo what should be the optimal refresh time?
	t := time.NewTicker(1 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:

		case <-r.vm.StopChan():
			return
		}
	}
}

// you must hold [r.l] when calling this function
func (r *RelayManager) request(
	ctx context.Context,
	j *relayJob,
) error {
	// @todo inform other relayers in network that these are active and relaying the data

	return nil
}

func (r *RelayManager) AppRequest(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	request []byte,
) error {
	// incoming requests from other nodes.
	reader := codec.NewReader(request, consts.MaxInt)
	relayerID := reader.UnpackInt(true)
	var msg []byte
	reader.UnpackBytes(-1, false, &msg)
	// send the request to relayer
	if err := r.serverless.SendToClient(relayerID, msg); err != nil {
		r.vm.Logger().Info("serverless error: %s", zap.Error(err))
		// don't send any response, if server did not exist.
	}
	return nil
}

func (r *RelayManager) HandleResponse(requestID uint32, response []byte) error {
	// @todo do if we get a response back
	return nil
}

func (r *RelayManager) HandleRequestFailed(requestID uint32) error {
	// @todo do if we get a failed request response
	return nil
}

func (r *RelayManager) Done() {
	<-r.done
}

func (r *RelayManager) SendRequest(
	ctx context.Context,
	data []byte,
) error {
	return nil
}
