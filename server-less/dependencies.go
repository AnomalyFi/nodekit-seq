package serverless

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
)

type RelayManager interface {
	SendRequestToAll(context.Context, int, []byte) error
	SendRequestToIndividual(context.Context, int, ids.NodeID, []byte) error
	SignAndSendRequestToIndividual(context.Context, int, []byte) error
}
