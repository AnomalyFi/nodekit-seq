package serverless

import (
	"github.com/ava-labs/avalanchego/ids"
)

var (
	SendToPeersMode             = byte(0)
	SendToValidatorMode         = byte(1)
	SignAndSendToPeersMode      = byte(2)
	SignAndSendToValidatorsMode = byte(3)
)

type SendToClientData struct {
	NodeID ids.NodeID `json:"node_id"`
	Data   []byte     `json:"data"`
}

type SendToPeersData struct {
	RelayerID int    `json:"relayer_id"`
	RawData   []byte `json:"raw_data"`
}

type SendToValidatorsData struct {
	RelayerID int        `json:"relayer_id"`
	NodeID    ids.NodeID `json:"node_id"`
	RawData   []byte     `json:"raw_data"`
}

type SignAndSendToPeersData struct {
	RelayerID          int    `json:"relayer_id"`
	IdentificationByte byte   `json:"identification_byte"`
	MsgBytes           []byte `json:"msg_bytes"`
}

type SignAndSendToValidatorData struct {
	RelayerID          int        `json:"relayer_id"`
	NodeID             ids.NodeID `json:"node_id"`
	IdentificationByte byte       `json:"identification_byte"`
	MsgBytes           []byte     `json:"msg_bytes"`
}

type SignedMessage struct {
	PublicKeyBytes       []byte `json:"publicKeyBytes"`
	SignatureBytes       []byte `json:"signature"`
	UnsignedMessageBytes []byte `json:"unsignedMessage"`
}
