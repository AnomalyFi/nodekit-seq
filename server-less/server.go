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
	SignAndSendRequestToIndividual func(context.Context, int, ids.NodeID, byte, []byte) error
	SignAndSendRequestToAll        func(context.Context, int, byte, []byte) error
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
		s.logger.Info("Received message", zap.Int("size", len(data)))
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
		case SendToValidatorMode:
			s.SendToValidator(data[1:])
		case SignAndSendToPeersMode:
			s.SignAndSendToPeers(data[1:])
		case SignAndSendToPeersMode:
			s.SignAndSendToValidator(data[1:])
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
		s.logger.Info("clients", zap.Any("clients", s.Clients))
		return fmt.Errorf("relayer does not exist with Id: %d", relayerID)
	}
	s.logger.Info("Sending data to client", zap.Int("size", len(data)))
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
	var stpd SendToPeersData
	err := json.Unmarshal(data, &stpd)
	if err != nil {
		return err
	}
	// send data to all validators
	if err := s.SendRequestToAll(context.Background(), stpd.RelayerID, stpd.RawData); err != nil {
		return err
	}
	return nil
}

// send data to a validator
func (s *ServerLess) SendToValidator(data []byte) error {
	var stvd SendToValidatorsData
	err := json.Unmarshal(data, &stvd)
	if err != nil {
		return err
	}
	if err := s.SendRequestToIndividual(context.Background(), stvd.RelayerID, stvd.NodeID, stvd.RawData); err != nil {
		return err
	}
	return nil
}

// note: for sign mode, the relayer side identification byte should be sent together with data
// sign and send data to all the validators
func (s *ServerLess) SignAndSendToPeers(data []byte) error {
	var saspd SignAndSendToPeersData
	err := json.Unmarshal(data, &saspd)
	if err != nil {
		return err
	}
	if err := s.SignAndSendRequestToAll(context.Background(), saspd.RelayerID, saspd.IdentificationByte, saspd.MsgBytes); err != nil {
		return err
	}
	return nil
}

// sign and send data to a validator
func (s *ServerLess) SignAndSendToValidator(data []byte) error {
	var sasvd SignAndSendToValidatorData
	err := json.Unmarshal(data, &sasvd)
	if err != nil {
		return err
	}
	if err := s.SignAndSendRequestToIndividual(context.Background(), sasvd.RelayerID, sasvd.NodeID, sasvd.IdentificationByte, sasvd.MsgBytes); err != nil {
		return err
	}
	return nil
}

// start the websocket server for bi-directional communication between relayers and node.
// it is on the relayers to validate the data they recieve from peers and to send valid data to peers.
func (s *ServerLess) Serverless(relayManager RelayManager, port string) {
	s.SendRequestToAll = relayManager.SendRequestToAll
	s.SendRequestToIndividual = relayManager.SendRequestToIndividual
	s.SignAndSendRequestToAll = relayManager.SignAndSendRequestToAll
	s.SignAndSendRequestToIndividual = relayManager.SignAndSendRequestToIndividual
	http.HandleFunc("/register-relayer", s.registerRelayer)
	http.ListenAndServe(port, nil)
	s.logger.Info("Server started", zap.String("port", port))
}
