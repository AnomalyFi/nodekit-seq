package types

import (
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	ToBChainID     = "tob"
	ToBBlockNumber = 0

	CertInfoSizeLimit = 2048
)

type DACertInfo struct {
	ChainID     string `json:"chainID"`
	BlockNumber uint64 `json:"blockNumber"` // for tob, this is the tob nonce
	Epoch       uint64 `json:"epoch"`
	ToBNonce    uint64 `json:"tobNonce"`
	ChunkID     ids.ID `json:"chunkID"`
	Cert        []byte `json:"cert"`
}

func (info *DACertInfo) Size() int {
	return codec.StringLen(info.ChainID) + 3*consts.Uint64Len + ids.IDLen + codec.BytesLen(info.Cert)
}

func (info *DACertInfo) Marshal(p *codec.Packer) {
	p.PackString(info.ChainID)
	p.PackUint64(info.BlockNumber)
	p.PackUint64(info.ToBNonce)
	p.PackUint64(info.Epoch)
	p.PackID(info.ChunkID)
	p.PackBytes(info.Cert)
}

func UnmarshalCertInfo(p *codec.Packer) (*DACertInfo, error) {
	ret := new(DACertInfo)
	ret.ChainID = p.UnpackString(true)
	ret.BlockNumber = p.UnpackUint64(false)
	ret.ToBNonce = p.UnpackUint64(false)
	ret.Epoch = p.UnpackUint64(false)
	p.UnpackID(true, &ret.ChunkID)
	p.UnpackBytes(CertInfoSizeLimit, true, &ret.Cert)
	if err := p.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}
