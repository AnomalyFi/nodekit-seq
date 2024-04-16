package controller

import (
	"context"
	"sync"

	"github.com/AnomalyFi/hypersdk/heap"
	"github.com/AnomalyFi/hypersdk/vm"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/engine/common"
)

type RelayManager struct {
	vm        *vm.VM
	appSender common.AppSender
	l         sync.Mutex
	requestID uint32

	pendingJobs *heap.Heap[*relayJob, int64]
	jobs        map[uint32]*relayJob
	done        chan struct{}
}

type relayJob struct {
}

func NewRelayManager(vm *vm.VM) *RelayManager {
	return &RelayManager{
		vm:          vm,
		pendingJobs: heap.New[*relayJob, int64](64, true),
		jobs:        make(map[uint32]*relayJob),
		done:        make(chan struct{}),
	}
}

func (b *RelayManager) Run(appSender common.AppSender) {

}

// you must hold [b.l] when calling this function
func (b *RelayManager) request(
	ctx context.Context,
	j *relayJob,
) error {
	// @todo load later
	return nil
}

func (b *RelayManager) AppRequest(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	request []byte,
) error {
	// @todo load later
	return nil
}

func (r *RelayManager) HandleResponse(requestID uint32, response []byte) error {
	// @todo load later
	return nil
}

func (r *RelayManager) HandleRequestFailed(requestID uint32) error {
	// @todo load later
	return nil
}
func (r *RelayManager) Done() {
	<-r.done
}

func (r *RelayManager) Broadcast(
	ctx context.Context,
	imageID ids.ID,
	proofValType uint16,
	chunkIndex uint16,
	data []byte,
) {

}
