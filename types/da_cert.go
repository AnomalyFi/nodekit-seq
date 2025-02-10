package types

import (
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
)

const (
	ToBChainID     = "tob"
	ToBBlockNumber = 0

	CertInfoSizeLimit = 2048
)

type DACertInfo struct {
	ChainID     string `json:"chainID"`
	BlockNumber uint64 `json:"blockNumber"` // for tob, this is the tob nonce
	Cert        []byte `json:"cert"`
}

func (info *DACertInfo) Size() int {
	return codec.StringLen(info.ChainID) + consts.Uint64Len + codec.BytesLen(info.Cert)
}

func (info *DACertInfo) Marshal(p *codec.Packer) error {
	p.PackString(info.ChainID)
	p.PackUint64(info.BlockNumber)
	p.PackBytes(info.Cert)

	return p.Err()
}

func UnmarshalCertInfo(p *codec.Packer) (*DACertInfo, error) {
	ret := new(DACertInfo)
	ret.ChainID = p.UnpackString(true)
	ret.BlockNumber = p.UnpackUint64(false)
	p.UnpackBytes(CertInfoSizeLimit, true, &ret.Cert)
	if err := p.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}
