package rpc

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/pubsub"
	"github.com/ava-labs/avalanchego/utils/logging"
	"go.uber.org/zap"
)

type WebSocketServer struct {
	logger logging.Logger
	s      *pubsub.Server
	// every verification contract will have its own commitment and verify attestation function
	// when ever a block containing seq-wasm transaction type is accepted, send all relevent transactions to the listeners
	seqWasmListeners *pubsub.Connections
}

func NewWebSocketServer(c Controller, maxPendingMessages int) (*WebSocketServer, *pubsub.Server) {
	w := &WebSocketServer{
		logger:           c.Logger(),
		seqWasmListeners: pubsub.NewConnections(),
	}
	cfg := pubsub.NewDefaultServerConfig()
	cfg.MaxPendingMessages = maxPendingMessages
	w.s = pubsub.New(w.logger, cfg, w.MessageCallback())
	return w, w.s
}

func (w *WebSocketServer) AcceptBlockWithSEQWasmTxs(b *chain.StatelessBlock) error {
	if w.seqWasmListeners.Len() > 0 {
		bytes, err := PackSEQWasmTxs(b)
		if err != nil {
			return err
		}
		inactiveConnection := w.s.Publish(append([]byte{seqWasmMode}, bytes...), w.seqWasmListeners)
		for _, conn := range inactiveConnection {
			w.seqWasmListeners.Remove(conn)
		}
	}
	return nil
}

func (w *WebSocketServer) MessageCallback() pubsub.Callback {
	var log = w.logger
	return func(msgBytes []byte, c *pubsub.Connection) {
		if len(msgBytes) == 0 {
			log.Error("failed to unmarshal msg", zap.Int("len", len(msgBytes)))
		}
		switch msgBytes[0] {
		case seqWasmMode:
			w.seqWasmListeners.Add(c)
			log.Debug("added seq wasm listener")
		default:
			log.Error("unexpected message type", zap.Int("len", len(msgBytes)), zap.Uint8("mode", msgBytes[0]))
		}
	}
}
