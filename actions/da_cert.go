package actions

import (
	"context"
	"fmt"

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

const (
	EigenDA = iota
)

var _ chain.Action = (*DACertificate)(nil)

type DACertificate struct {
	Cert *types.DACertInfo `json:"cert"`
}

func (*DACertificate) GetTypeID() uint8 {
	return DACertID
}

func (cert *DACertificate) StateKeys(_ codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.DACertToBNonceKey()):                                        state.All,
		string(storage.ToBNonceEpochKey(cert.Cert.Epoch)):                          state.All,
		string(storage.DACertChunkIDKey(cert.Cert.ChainID, cert.Cert.BlockNumber)): state.All,
		string(storage.DACertIndexKey(cert.Cert.ToBNonce)):                         state.All,
		string(storage.DACertByChunkIDKey(cert.Cert.ChunkID)):                      state.All,
	}
}

func (*DACertificate) StateKeysMaxChunks() []uint16 {
	return []uint16{
		storage.DACertificateChunks, storage.DACertificateChunks,
	}
}

func (cert *DACertificate) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	_ uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	if !isWhiteListed(rules, actor) {
		return nil, ErrNotWhiteListed
	}

	// store highest ToBNonce if there's new one
	tobNonce, err := storage.GetDACertToBNonce(ctx, mu)
	if err != nil {
		return nil, fmt.Errorf("failed to get tob nonce: %w", err)
	}
	if cert.Cert.ToBNonce > tobNonce {
		if err := storage.SetDACertToBNonce(ctx, mu, tobNonce); err != nil {
			return nil, fmt.Errorf("failed to store tob nonce: %w", err)
		}
	}

	// add current chunk to chunk layer at ToBNonce
	if err := storage.AddDACertToLayer(ctx, mu, cert.Cert.ToBNonce, cert.Cert.ChunkID); err != nil {
		return nil, fmt.Errorf("failed to add cert chunk layer: %w", err)
	}

	// store the cert and the index
	if err := storage.SetDACertChunkID(ctx, mu, cert.Cert.ChainID, cert.Cert.BlockNumber, cert.Cert.ChunkID); err != nil {
		return nil, fmt.Errorf("failed to set cert chunk id: %w", err)
	}

	if err := storage.SetDACert(ctx, mu, cert.Cert.ChunkID, cert.Cert); err != nil {
		return nil, fmt.Errorf("failed to save cert: %w", err)
	}
	// set the lowest ToBNonce for the epoch
	if err := storage.SetToBNonceAtEpoch(ctx, mu, cert.Cert.Epoch, cert.Cert.ToBNonce); err != nil {
		return nil, fmt.Errorf("failed to set tobnonce of epoch: %w", err)
	}

	return nil, nil
}

func (*DACertificate) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return DACertComputeUnits
}

func (cert *DACertificate) Size() int {
	return consts.ByteLen + cert.Cert.Size()
}

func (cert *DACertificate) Marshal(p *codec.Packer) {
	cert.Cert.Marshal(p)
}

func UnmarshalDACertificate(p *codec.Packer) (chain.Action, error) {
	var cert DACertificate
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

func (*DACertificate) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*DACertificate) UseFeeMarket() bool {
	return true
}
