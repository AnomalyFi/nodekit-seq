package serverless

type SendToPeersData struct {
	RawData   []byte `json:"raw_data"`
	RelayerID int    `json:"relayer_id"`
}
