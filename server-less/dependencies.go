package serverless

import "context"

type RelayManager interface {
	SendRequestToAll(context.Context, int, []byte) error
}
