package serverless

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	// "github.com/AnomalyFi/nodekit-seq/controller"
	"github.com/gorilla/websocket"
)

type ServerLess struct {
	Clients     map[int]*websocket.Conn
	Upgrader    websocket.Upgrader
	clientsL    sync.Mutex
	SendRequest func(context.Context, []byte) error
}

func NewServerLess(readBufferSize, writeBufferSize int) *ServerLess {
	return &ServerLess{
		Clients: make(map[int]*websocket.Conn),
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  readBufferSize,
			WriteBufferSize: writeBufferSize,
		},
	}
}

func (s *ServerLess) registerRelayer(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := s.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// Read the identification number from the client
	var realyerID int
	err = conn.ReadJSON(&realyerID)
	if err != nil {
		log.Println(err)
		return
	}
	err = conn.WriteMessage(websocket.TextMessage, []byte("connected"))
	if err != nil {
		log.Println(err)
	}
	s.clientsL.Lock()
	s.Clients[realyerID] = conn
	s.clientsL.Unlock()
}

func (s *ServerLess) SendToClient(relayerID int, data []byte) error {
	// send data to client/conn
	s.clientsL.Lock()
	conn, ok := s.Clients[relayerID]
	s.clientsL.Unlock()
	if !ok {
		return fmt.Errorf("relayer does not exist for this validator with relayerId: %d", relayerID)
	}

	err := conn.WriteMessage(websocket.BinaryMessage, data) // make differentiability
	if err != nil {
		s.clientsL.Lock()
		delete(s.Clients, relayerID)
		s.clientsL.Unlock()
		return fmt.Errorf("can't send request to client: %s", err)
	}
	return nil
}

func (s *ServerLess) SendToPeers(w http.ResponseWriter, r *http.Request) {
	// send data to all peers
	// data can be anything arbitary.
	s.SendRequest(context.TODO(), []byte{})
}

func (s *ServerLess) Settle(w http.ResponseWriter, r *http.Request) {
	// sends the signed signature only to the validator realyer who successfully satisfied relay request

}

// it is on the individual relayers to keep track of all the nodeID's realted requests
func (s *ServerLess) Serverless(controller Controller) {
	s.SendRequest = controller.SendRequest
	//@todo add all handlers for bidirectional communication
	http.HandleFunc("/register-relayer", s.registerRelayer)
	http.HandleFunc("/send", s.SendToPeers)
	http.HandleFunc("/settle", s.Settle)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
