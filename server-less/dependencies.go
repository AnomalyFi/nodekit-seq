package serverless

import "context"

type Controller interface {
	SendRequest(context.Context, []byte) error
}
