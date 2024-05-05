package serverless

type SendToPeersData struct {
	RawData   []byte `json:"raw_data"`
	RelayerID int    `json:"relayer_id"`
}

var (
	sendToPeersMode = byte(0)
	settleMode      = byte(1)
)
