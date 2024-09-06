// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/fees"
	seqconsts "github.com/AnomalyFi/nodekit-seq/consts"

	"github.com/AnomalyFi/hypersdk/codec"

	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/types"
)

type JSONRPCServer struct {
	c Controller
}

func NewJSONRPCServer(c Controller) *JSONRPCServer {
	return &JSONRPCServer{
		c: c,
	}
}

type GenesisReply struct {
	Genesis *genesis.Genesis `json:"genesis"`
}

func (j *JSONRPCServer) Genesis(_ *http.Request, _ *struct{}, reply *GenesisReply) (err error) {
	reply.Genesis = j.c.Genesis()
	return nil
}

type TxArgs struct {
	TxID ids.ID `json:"txId"`
}

type TxReply struct {
	Timestamp int64           `json:"timestamp"`
	Success   bool            `json:"success"`
	Units     fees.Dimensions `json:"units"`
	Fee       uint64          `json:"fee"`
}

func (j *JSONRPCServer) Tx(req *http.Request, args *TxArgs, reply *TxReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Tx")
	defer span.End()

	found, t, success, units, fee, err := j.c.GetTransaction(ctx, args.TxID)
	if err != nil {
		return err
	}
	if !found {
		return ErrTxNotFound
	}
	reply.Timestamp = t
	reply.Success = success
	reply.Units = units
	reply.Fee = fee
	return nil
}

type AssetArgs struct {
	Asset ids.ID `json:"asset"`
}

type AssetReply struct {
	Symbol   []byte `json:"symbol"`
	Decimals uint8  `json:"decimals"`
	Metadata []byte `json:"metadata"`
	Supply   uint64 `json:"supply"`
	Owner    string `json:"owner"`
}

func (j *JSONRPCServer) Asset(req *http.Request, args *AssetArgs, reply *AssetReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Asset")
	defer span.End()

	exists, symbol, decimals, metadata, supply, owner, err := j.c.GetAssetFromState(ctx, args.Asset)
	if err != nil {
		return err
	}
	if !exists {
		return ErrAssetNotFound
	}
	reply.Symbol = symbol
	reply.Decimals = decimals
	reply.Metadata = metadata
	reply.Supply = supply
	reply.Owner = codec.MustAddressBech32(seqconsts.HRP, owner)
	return err
}

type BalanceArgs struct {
	Address string `json:"address"`
	Asset   ids.ID `json:"asset"`
}

type BalanceReply struct {
	Amount uint64 `json:"amount"`
}

func (j *JSONRPCServer) Balance(req *http.Request, args *BalanceArgs, reply *BalanceReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Balance")
	defer span.End()

	addr, err := codec.ParseAddressBech32(seqconsts.HRP, args.Address)
	if err != nil {
		return err
	}
	balance, err := j.c.GetBalanceFromState(ctx, addr, args.Asset)
	if err != nil {
		return err
	}
	reply.Amount = balance
	return err
}

type BlockInfo struct {
	BlockId   string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	L1Head    uint64 `json:"l1_head"`
	Height    uint64 `json:"height"`
}

type BlockHeadersResponse struct {
	From   uint64      `json:"from"`
	Blocks []BlockInfo `json:"blocks"`
	Prev   BlockInfo   `json:"prev"`
	Next   BlockInfo   `json:"next"`
}

// TODO need to fix this. Tech debt
type TransactionResponse struct {
	Txs     []*chain.Transaction `json:"txs"`
	BlockId string               `json:"id"`
}

type SEQTransactionResponse struct {
	Txs     []*types.SEQTransaction `json:"txs"`
	BlockId string                  `json:"id"`
}

type SequencerWarpBlockResponse struct {
	Blocks []SequencerWarpBlock `json:"blocks"`
}

type SequencerWarpBlock struct {
	BlockId    string   `json:"id"`
	Timestamp  int64    `json:"timestamp"`
	L1Head     uint64   `json:"l1_head"`
	Height     *big.Int `json:"height"`
	BlockRoot  *big.Int `json:"root"`
	ParentRoot *big.Int `json:"parent"`
}

type GetBlockHeadersByHeightArgs struct {
	Height uint64 `json:"height"`
	End    int64  `json:"end"`
}

type GetBlockHeadersIDArgs struct {
	ID  string `json:"id"`
	End int64  `json:"end"`
}

type GetBlockHeadersByStartArgs struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type GetBlockTransactionsArgs struct {
	ID string `json:"block_id"`
}

type GetBlockCommitmentArgs struct {
	First         uint64 `json:"first"`
	CurrentHeight uint64 `json:"current_height"`
	MaxBlocks     int    `json:"max_blocks"`
}

type GetBlockTransactionsByNamespaceArgs struct {
	Height    uint64 `json:"height"`
	Namespace string `json:"namespace"`
}

func (j *JSONRPCServer) GetBlockHeadersByHeight(req *http.Request, args *types.GetBlockHeadersByHeightArgs, reply *types.BlockHeadersResponse) error {
	headers, err := j.c.Archiver().GetBlockHeadersByHeight(args)
	if err != nil {
		return err
	}

	reply.From = headers.From
	reply.Blocks = headers.Blocks
	reply.Prev = headers.Prev
	reply.Next = headers.Next

	return nil
}

func (j *JSONRPCServer) GetBlockHeadersID(req *http.Request, args *types.GetBlockHeadersIDArgs, reply *types.BlockHeadersResponse) error {
	headers, err := j.c.Archiver().GetBlockHeadersByID(args)
	if err != nil {
		return err
	}

	reply.From = headers.From
	reply.Blocks = headers.Blocks
	reply.Prev = headers.Prev
	reply.Next = headers.Next

	return nil
}

func (j *JSONRPCServer) GetBlockHeadersByStart(req *http.Request, args *types.GetBlockHeadersByStartArgs, reply *types.BlockHeadersResponse) error {
	// Parse query parameters
	headers, err := j.c.Archiver().GetBlockHeadersAfterTimestamp(args)
	if err != nil {
		return err
	}

	reply.From = headers.From
	reply.Blocks = headers.Blocks
	reply.Prev = headers.Prev
	reply.Next = headers.Next

	return nil
}

func (j *JSONRPCServer) GetBlockTransactions(req *http.Request, args *GetBlockTransactionsArgs, reply *types.SEQTransactionResponse) error {
	// Parse query parameters

	// TODO either the firstBlock height is equal to height or use the hash to get it or if none of the above work then use the btree to get it
	if args.ID == "" {
		return fmt.Errorf("block id not provided")
	}

	parser := j.ServerParser(req.Context(), j.c.NetworkID(), j.c.ChainID())

	blk, err := j.c.Archiver().GetBlockByID(args.ID, parser)
	if err != nil {
		return err
	}

	// only append sequencer msg actions
	for _, tx := range blk.Txs {
		for k, action := range tx.Actions {
			switch action := action.(type) {
			case *actions.SequencerMsg:
				ns := hex.EncodeToString(action.NMTNamespace())
				reply.Txs = append(reply.Txs, &types.SEQTransaction{
					Namespace:   ns,
					Transaction: action.Data, // eth format tx binary
					Index:       uint64(k),
					Tx_id:       tx.ID().String(),
				})
			}
		}
	}

	reply.BlockId = args.ID

	return nil
}

func (j *JSONRPCServer) GetCommitmentBlocks(req *http.Request, args *types.GetBlockCommitmentArgs, reply *types.SequencerWarpBlockResponse) error {
	parser := j.ServerParser(req.Context(), j.c.NetworkID(), j.c.ChainID())
	warpResp, err := j.c.Archiver().GetCommitmentBlocks(args, parser)
	if err != nil {
		return err
	}
	reply.Blocks = warpResp.Blocks

	return nil
}

func (j *JSONRPCServer) GetBlockTransactionsByNamespace(req *http.Request, args *GetBlockTransactionsByNamespaceArgs, reply *SEQTransactionResponse) error {
	// TODO either the firstBlock height is equal to height or use the hash to get it or if none of the above work then use the btree to get it
	parser := j.ServerParser(req.Context(), j.c.NetworkID(), j.c.ChainID())

	blk, err := j.c.Archiver().GetBlockByHeight(args.Height, parser)
	if err != nil {
		return err
	}
	blkID, err := blk.ID()
	if err != nil {
		return err
	}

	// only append sequencer msg actions
	for _, tx := range blk.Txs {
		for k, action := range tx.Actions {
			switch action := action.(type) {
			case *actions.SequencerMsg:
				ns := hex.EncodeToString(action.NMTNamespace())
				if args.Namespace != ns {
					continue
				}
				reply.Txs = append(reply.Txs, &types.SEQTransaction{
					Namespace:   ns,
					Transaction: action.Data, // eth format tx binary
					Index:       uint64(k),   // might be duplicate
					Tx_id:       tx.ID().String(),
				})
			}
		}
	}

	reply.BlockId = blkID.String()

	return nil
}

var _ chain.Parser = (*ServerParser)(nil)

type ServerParser struct {
	networkID uint32
	chainID   ids.ID
	genesis   *genesis.Genesis
}

func (p *ServerParser) ChainID() ids.ID {
	return p.chainID
}

func (p *ServerParser) Rules(t int64) chain.Rules {
	return p.genesis.Rules(t, p.networkID, p.chainID)
}

func (*ServerParser) Registry() (chain.ActionRegistry, chain.AuthRegistry) {
	return seqconsts.ActionRegistry, seqconsts.AuthRegistry
}

func (j *JSONRPCServer) ServerParser(ctx context.Context, networkId uint32, chainId ids.ID) chain.Parser {
	g := j.c.Genesis()

	// The only thing this is using is the ActionRegistry and AuthRegistry so this should be fine
	return &Parser{networkId, chainId, g}
}

func (j *JSONRPCServer) GetAcceptedBlockWindow(req *http.Request, _ *struct{}, reply *int) error {
	*reply = j.c.GetAcceptedBlockWindow()
	return nil
}
