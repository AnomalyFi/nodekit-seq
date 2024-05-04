package serverless

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type ServerLess struct {
	Clients          map[int]*websocket.Conn
	Upgrader         websocket.Upgrader
	clientsL         sync.Mutex
	SendRequestToAll func(context.Context, int, []byte) error
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

// register a new relayer or update the existing relayer
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
	// add/update relayer
	s.clientsL.Lock()
	s.Clients[realyerID] = conn
	s.clientsL.Unlock()
}

// @todo lets see, how the both below methods work

// send data received from peer to the client relayer
func (s *ServerLess) SendToClient(relayerID int, data []byte) error {
	s.clientsL.Lock()
	conn, ok := s.Clients[relayerID]
	s.clientsL.Unlock()
	if !ok {
		return fmt.Errorf("relayer does not exist with Id: %d", relayerID)
	}

	err := conn.WriteMessage(websocket.BinaryMessage, data)
	if err != nil {
		s.clientsL.Lock()
		delete(s.Clients, relayerID)
		s.clientsL.Unlock()
		return fmt.Errorf("can't send request to client: %s", err)
	}
	return nil
}

// send data to all the validators
func (s *ServerLess) SendToPeers(w http.ResponseWriter, r *http.Request) {
	var rawData SendToPeersData
	err := json.NewDecoder(r.Body).Decode(&rawData)
	defer r.Body.Close()
	if err != nil { //@todo send better error status code and messages
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	// data unmarshalled successfully, send it to all the validators
	if err := s.SendRequestToAll(context.Background(), rawData.RelayerID, rawData.RawData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

// send signature only to the validator, that satisfied the relay request
func (s *ServerLess) Settle(w http.ResponseWriter, r *http.Request) {

}

// start the websocker server for bi-directional communication between relayers and node.
// it is on the relayers to validate the data they recieve from peers and to send valid data to peers.
func (s *ServerLess) Serverless(relayManager RelayManager, port string) {
	s.SendRequestToAll = relayManager.SendRequestToAll
	http.HandleFunc("/register-relayer", s.registerRelayer)
	http.HandleFunc("/send", s.SendToPeers)
	http.HandleFunc("/settle", s.Settle)
	log.Fatal(http.ListenAndServe(port, nil))
}
