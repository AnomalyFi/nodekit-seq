// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/crypto"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/ava-labs/avalanchego/ids"
)

//TODO need to fix the first 2 things because those will not work for the byte array. I need to figure out the max len of byte array
// We restrict the [MaxContentSize] to be 768B such that any collection of
// 1024 keys+values will never be more than [chain.NetworkSizeLimit].
//
// TODO: relax this once merkleDB/sync can ensure range proof resolution does
// not surpass [NetworkSizeLimit]
const WarpSequencerMsgSize = 1_600

// consts.IDLen + consts.IDLen +
// 	crypto.PublicKeyLen + consts.Uint64Len + consts.Uint64Len +
// 	consts.IDLen + consts.Uint64Len + consts.Uint64Len + consts.IDLen
//TODO Do I need the FromAddress?
type WarpSequencerMsg struct {
	ChainId     []byte           `protobuf:"bytes,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	Data        []byte           `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	FromAddress crypto.PublicKey `json:"from_address"`

	// SwapExpiry is the unix timestamp at which the swap becomes invalid (and
	// the message can be processed without a swap.
	SwapExpiry int64 `json:"swapExpiry"`

	// TxID is the transaction that created this message. This is used to ensure
	// there is WarpID uniqueness.
	TxID ids.ID `json:"txID"`
}

func (w *WarpSequencerMsg) Marshal() ([]byte, error) {
	p := codec.NewWriter(WarpTransferSize)
	p.PackBytes(w.Data)
	p.PackBytes(w.ChainId)
	p.PackPublicKey(w.FromAddress)
	op := codec.NewOptionalWriter()
	op.PackInt64(w.SwapExpiry)
	p.PackOptional(op)
	p.PackID(w.TxID)
	return p.Bytes(), p.Err()
}

func ImportedMsgID(sourceChainID ids.ID) ids.ID {
	return utils.ToID(ImportedMsgMetadata(sourceChainID))
}

func ImportedMsgMetadata(sourceChainID ids.ID) []byte {
	k := make([]byte, consts.IDLen)
	copy(k[consts.IDLen:], sourceChainID[:])
	return k
}

func UnmarshalWarpSequencerMsg(b []byte) (*WarpSequencerMsg, error) {
	var msg WarpSequencerMsg
	p := codec.NewReader(b, WarpSequencerMsgSize)
	p.UnpackBytes(-1, true, &msg.Data)
	p.UnpackBytes(-1, true, &msg.ChainId)
	p.UnpackPublicKey(false, &msg.FromAddress)
	op := p.NewOptionalReader()
	msg.SwapExpiry = op.UnpackInt64()
	op.Done()
	p.UnpackID(true, &msg.TxID)
	if err := p.Err(); err != nil {
		return nil, err
	}
	if !p.Empty() {
		return nil, chain.ErrInvalidObject
	}
	// Handle swap checks
	if msg.SwapExpiry < 0 {
		return nil, chain.ErrInvalidObject
	}
	return &msg, nil
}
