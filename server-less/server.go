package serverless

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type ServerLess struct {
	Clients                        map[int]*websocket.Conn
	Upgrader                       websocket.Upgrader
	clientsL                       sync.Mutex
	logger                         logging.Logger
	SendRequestToAll               func(context.Context, int, []byte) error
	SendRequestToIndividual        func(context.Context, int, ids.NodeID, []byte) error
	SignAndSendRequestToIndividual func(context.Context, int, []byte) error
}

func NewServerLess(readBufferSize, writeBufferSize int, logger logging.Logger) *ServerLess {
	return &ServerLess{
		Clients: make(map[int]*websocket.Conn),
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  readBufferSize,
			WriteBufferSize: writeBufferSize,
		},
		logger: logger,
	}
}

// register a new relayer or update the existing relayer.
func (s *ServerLess) registerRelayer(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade the connection", zap.Error(err))
		return
	}
	defer conn.Close()
	// Read the identification number from the client
	var realyerID int
	err = conn.ReadJSON(&realyerID)
	if err != nil {
		s.logger.Error("Failed to read the relayer ID", zap.Error(err))
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte("connected"))
	if err != nil {
		s.logger.Error("Failed to send the connection confirmation", zap.Error(err))
	}
	// add/update relayer
	s.clientsL.Lock()
	s.Clients[realyerID] = conn
	s.clientsL.Unlock()
	// handle the messages from the client
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("Failed to read the message", zap.Error(err))
			}
			break
		}
		if len(data) > 100*1024 {
			s.logger.Error("Data is too large", zap.Int("size", len(data)), zap.Int("max size", 100*1024))
			continue
		}
		switch data[0] {
		case SendToPeersMode:
			s.SendToPeers(data[1:])
		case SettleMode:
			s.Settle(data[1:])
		default:
			s.logger.Error("Unknown mode", zap.Int("mode", int(data[0])))
		}
	}
}

// send data received from peer to the client relayer
func (s *ServerLess) SendToClient(relayerID int, nodeID ids.NodeID, data []byte) error {
	s.clientsL.Lock()
	conn, ok := s.Clients[relayerID]
	s.clientsL.Unlock()
	if !ok {
		return fmt.Errorf("relayer does not exist with Id: %d", relayerID)
	}
	err := conn.WriteJSON(SendToClientData{nodeID, data})
	if err != nil {
		s.clientsL.Lock()
		delete(s.Clients, relayerID)
		s.clientsL.Unlock()
		return fmt.Errorf("can't send request to client: %s", err)
	}
	return nil
}

// send data to all the validators
func (s *ServerLess) SendToPeers(data []byte) error {
	var rawData SendToPeersData
	err := json.Unmarshal(data, &rawData)
	if err != nil {
		return err
	}
	// send data to validators
	if err := s.SendRequestToAll(context.Background(), rawData.RelayerID, rawData.RawData); err != nil {
		return err
	}
	return nil
}

func (s *ServerLess) Settle(data []byte) error {
	var sasd SignAndSettleData
	err := json.Unmarshal(data, &sasd)
	if err != nil {
		return err
	}
	if err := s.SignAndSendRequestToIndividual(context.Background(), sasd.RelayerID, sasd.MsgBytes); err != nil {
		return err
	}
	return nil
}

// start the websocker server for bi-directional communication between relayers and node.
// it is on the relayers to validate the data they recieve from peers and to send valid data to peers.
func (s *ServerLess) Serverless(relayManager RelayManager, port string) {
	s.SendRequestToAll = relayManager.SendRequestToAll
	s.SendRequestToIndividual = relayManager.SendRequestToIndividual
	s.SignAndSendRequestToIndividual = relayManager.SignAndSendRequestToIndividual
	http.HandleFunc("/register-relayer", s.registerRelayer)
	http.ListenAndServe(port, nil)
	s.logger.Info("Server started", zap.String("port", port))
}
