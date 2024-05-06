package serverless

import "github.com/ava-labs/avalanchego/ids"

type SendToPeersData struct {
	RawData   []byte `json:"raw_data"`
	RelayerID int    `json:"relayer_id"`
}

type SendToClientData struct {
	NodeID ids.NodeID `json:"node_id"`
	Data   []byte     `json:"data"`
}

var (
	sendToPeersMode = byte(0)
	settleMode      = byte(1)
)
