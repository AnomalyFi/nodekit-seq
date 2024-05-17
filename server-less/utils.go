package serverless

import (
	"github.com/ava-labs/avalanchego/ids"
)

type SendToPeersData struct {
	RawData   []byte `json:"raw_data"`
	RelayerID int    `json:"relayer_id"`
}

type SendToClientData struct {
	NodeID ids.NodeID `json:"node_id"`
	Data   []byte     `json:"data"`
}

var (
	SendToPeersMode = byte(0)
	SettleMode      = byte(1)
)

// validators will use this signed message to claim their relay fees, from SEQ.
type UnsignedMessage struct {
	RelayerID  int    `json:"relayerID"`
	StartBlock uint64 `json:"startBlock"`
	EndBlock   uint64 `json:"endBlock"`
}

type SignedMessage struct {
	PublicKeyBytes       []byte `json:"publicKeyBytes"`
	SignatureBytes       []byte `json:"signature"`
	UnsignedMessageBytes []byte `json:"unsignedMessage"`
}

type Message struct {
	NodeID          ids.NodeID      `json:"nodeID"`
	UnSignedMessage UnsignedMessage `json:"UnSignedMessage"`
}

type SignAndSettleData struct {
	RelayerID int    `json:"relayer_id"`
	MsgBytes  []byte `json:"msg_bytes"`
}
