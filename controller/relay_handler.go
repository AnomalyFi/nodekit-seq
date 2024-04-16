package controller

import (
	"context"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/version"
)

type RelayHandler struct {
	c *Controller
}

func NewRelayHandler(c *Controller) *RelayHandler {
	return &RelayHandler{c}
}

func (b *RelayHandler) Connected(
	ctx context.Context,
	nodeID ids.NodeID,
	v *version.Application,
) error {
	return nil
}

func (b *RelayHandler) Disconnected(ctx context.Context, nodeID ids.NodeID) error {
	return nil
}

func (*RelayHandler) AppGossip(context.Context, ids.NodeID, []byte) error {
	return nil
}

func (b *RelayHandler) AppRequest(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	_ time.Time,
	request []byte,
) error {

	return b.c.relayManager.AppRequest(ctx, nodeID, requestID, request)
}

func (b *RelayHandler) AppRequestFailed(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
) error {
	return b.c.relayManager.HandleRequestFailed(requestID)
}

func (b *RelayHandler) AppResponse(
	ctx context.Context,
	nodeID ids.NodeID,
	requestID uint32,
	response []byte,
) error {
	return b.c.relayManager.HandleResponse(requestID, response)
}

func (*RelayHandler) CrossChainAppRequest(
	context.Context,
	ids.ID,
	uint32,
	time.Time,
	[]byte,
) error {
	return nil
}

func (*RelayHandler) CrossChainAppRequestFailed(context.Context, ids.ID, uint32) error {
	return nil
}

func (*RelayHandler) CrossChainAppResponse(context.Context, ids.ID, uint32, []byte) error {
	return nil
}
