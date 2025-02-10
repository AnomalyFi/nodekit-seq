package actions

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/AnomalyFi/nodekit-seq/types"
	"github.com/ava-labs/avalanchego/ids"
)

const (
	CertLimit = 2048 // 2KB
)

var _ chain.Action = (*DACertificate)(nil)

type DACertificate struct {
	DAType   uint8             `json:"daID"`
	ChunkID  ids.ID            `json:"chunkID"`
	ToBNonce uint64            `json:"tobNonce"`
	Cert     *types.DACertInfo `json:"cert"`
}

func (*DACertificate) GetTypeID() uint8 {
	return MsgID
}

func (cert *DACertificate) StateKeys(_ codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.DACertToBNonceKey()):           state.All,
		string(storage.DACertIndexKey(cert.ToBNonce)): state.All,
		string(storage.DACertKey(cert.ChunkID)):       state.All,
	}
}

func (*DACertificate) StateKeysMaxChunks() []uint16 {
	return []uint16{
		storage.DACertificateChunks, storage.DACertificateChunks,
	}
}

func (cert *DACertificate) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	_ uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	tobNonce, err := storage.GetDACertToBNonce(ctx, mu)
	if err != nil {
		return nil, err
	}
	if cert.ToBNonce > tobNonce {
		if err := storage.SetDACertToBNonce(ctx, mu, tobNonce); err != nil {
			return nil, err
		}
	}

	if err := storage.AddDACertToLayer(ctx, mu, cert.ToBNonce, cert.ChunkID); err != nil {
		return nil, err
	}

	if err := storage.SetDACert(ctx, mu, cert.ChunkID, cert.Cert); err != nil {
		return nil, err
	}

	return nil, nil
}

func (*DACertificate) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return DACertComputeUnits
}

func (cert *DACertificate) Size() int {
	return consts.ByteLen + cert.Cert.Size() + ids.IDLen
}

func (cert *DACertificate) Marshal(p *codec.Packer) {
	p.PackByte(cert.DAType)
	p.PackID(cert.ChunkID)
	cert.Cert.Marshal(p)
}

func UnmarshalDACertificate(p *codec.Packer) (chain.Action, error) {
	var cert DACertificate
	cert.DAType = p.UnpackByte()
	p.UnpackID(true, &cert.ChunkID)
	certInfo, err := types.UnmarshalCertInfo(p)
	if err != nil {
		return nil, err
	}
	cert.Cert = certInfo
	return &cert, p.Err()
}

func (*DACertificate) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (cert *DACertificate) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*DACertificate) UseFeeMarket() bool {
	return true
}
