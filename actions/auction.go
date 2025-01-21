package actions

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/crypto/bls"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
)

var _ chain.Action = (*Auction)(nil)

type AuctionInfo struct {
	EpochNumber       uint64        `json:"epochNumber"`
	BidPrice          uint64        `json:"bidPrice"`
	BuilderSEQAddress codec.Address `json:"builderSEQAddress"`
}

func (info *AuctionInfo) HashAuctionInfo() ([32]byte, error) {
	msg := binary.BigEndian.AppendUint64(nil, info.EpochNumber)
	msg = binary.BigEndian.AppendUint64(msg, info.BidPrice)
	msg = append(msg, info.BuilderSEQAddress[:]...)
	hash := sha256.Sum256(msg)
	return hash, nil
}

func (info *AuctionInfo) Marshal(p *codec.Packer) {
	p.PackUint64(info.EpochNumber)
	p.PackUint64(info.BidPrice)
	p.PackAddress(info.BuilderSEQAddress)
}

func UnmarshalAuctionInfo(p *codec.Packer) (*AuctionInfo, error) {
	var auctionInfo AuctionInfo
	auctionInfo.EpochNumber = p.UnpackUint64(true)
	auctionInfo.BidPrice = p.UnpackUint64(true)
	p.UnpackAddress(&auctionInfo.BuilderSEQAddress)
	if p.Err() != nil {
		return nil, p.Err()
	}
	return &auctionInfo, nil
}

type Auction struct {
	AuctionInfo      AuctionInfo `json:"auctionInfo"`
	BuilderPublicKey []byte      `json:"builderPublicKey"` // BLS public key of the bidder.
	BuilderSignature []byte      `json:"signature"`
}

func (*Auction) GetTypeID() uint8 {
	return AuctionID
}

func (a *Auction) StateKeys(_ codec.Address, _ ids.ID) state.Keys {
	return state.Keys{
		string(storage.BalanceKey(a.AuctionInfo.BuilderSEQAddress)): state.Read | state.Write,
		string(storage.BalanceKey(ArcadiaFundAddress())):            state.All,
		string(storage.ArcadiaBidKey(a.AuctionInfo.EpochNumber)):    state.Write | state.Allocate,
	}
}

func (*Auction) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.BalanceChunks, storage.BalanceChunks, hactions.EpochExitsChunks}
}

// This is a permissioned action, only authorized address can pass the execution.
func (a *Auction) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	hght uint64,
	actor codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	// TODO: this allows any whitelisted address to submit arcadia bid.
	// change this to only allow the arcadia address to submit the bid.
	if !isWhiteListed(rules, actor) {
		return nil, ErrNotWhiteListed
	}

	// Allow auction bid for next epoch processed only during the current epoch.
	if a.AuctionInfo.EpochNumber != Epoch(hght, rules.GetEpochLength())+1 {
		return nil, fmt.Errorf("epoch number is not valid, expected: %d, actual: %d", Epoch(hght, rules.GetEpochLength())+1, a.AuctionInfo.EpochNumber)
	}

	pubkey, err := bls.PublicKeyFromBytes(a.BuilderPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	digest, err := a.AuctionInfo.HashAuctionInfo()
	if err != nil {
		return nil, fmt.Errorf("unable to hash auction info")
	}

	sig, err := bls.SignatureFromBytes(a.BuilderSignature)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signature: %w", err)
	}

	// Verify the signature.
	if !bls.Verify(digest[:], pubkey, sig) {
		return nil, ErrInvalidBidderSignature
	}

	builderSEQAddress := codec.CreateAddress(auth.BLSID, utils.ToID(a.BuilderPublicKey))
	if builderSEQAddress != a.AuctionInfo.BuilderSEQAddress {
		return nil, ErrParsedBuilderSEQAddressMismatch
	}
	// deduct the bid amount from bidder.
	if err := storage.SubBalance(ctx, mu, builderSEQAddress, a.AuctionInfo.BidPrice); err != nil {
		return nil, err
	}

	// Bid amount is sent to the Arcadia fund address.
	if err := storage.AddBalance(ctx, mu, ArcadiaFundAddress(), a.AuctionInfo.BidPrice, true); err != nil {
		return nil, err
	}

	// Store bid information.
	if err := storage.StoreArcadiaBidInfo(ctx, mu, a.AuctionInfo.EpochNumber, a.AuctionInfo.BidPrice, a.BuilderPublicKey, a.BuilderSignature); err != nil {
		return nil, err
	}

	return nil, nil
}

func (*Auction) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return AuctionComputeUnits
}

func (*Auction) Size() int {
	return 2*consts.Uint64Len + bls.PublicKeyLen + bls.SignatureLen + codec.AddressLen
}

func (a *Auction) Marshal(p *codec.Packer) {
	a.AuctionInfo.Marshal(p)
	p.PackBytes(a.BuilderPublicKey)
	p.PackBytes(a.BuilderSignature)
}

func UnmarshalAuction(p *codec.Packer) (chain.Action, error) {
	var auction Auction
	auctionInfo, err := UnmarshalAuctionInfo(p)
	if err != nil {
		return nil, err
	}
	auction.AuctionInfo = *auctionInfo
	p.UnpackBytes(48, true, &auction.BuilderPublicKey)
	p.UnpackBytes(96, true, &auction.BuilderSignature)
	return &auction, p.Err()
}

func (*Auction) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*Auction) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*Auction) UseFeeMarket() bool {
	return false
}
