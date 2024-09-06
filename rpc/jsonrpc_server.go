// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
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

type BalanceArgs struct {
	Address string `json:"address"`
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
	balance, err := j.c.GetBalanceFromState(ctx, addr)
	if err != nil {
		return err
	}
	reply.Amount = balance
	return err
}

func (j *JSONRPCServer) GetBlockHeadersByHeight(req *http.Request, args *types.GetBlockHeadersByHeightArgs, reply *types.BlockHeadersResponse) error {
	_, span := j.c.Tracer().Start(req.Context(), "Server.GetBlockHeadersByHeight")
	defer span.End()
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
	_, span := j.c.Tracer().Start(req.Context(), "Server.GetBlockHeadersID")
	defer span.End()

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

func (j *JSONRPCServer) GetBlockHeadersByStartTimeStamp(req *http.Request, args *types.GetBlockHeadersByStartTimeStampArgs, reply *types.BlockHeadersResponse) error {
	_, span := j.c.Tracer().Start(req.Context(), "Server.GetBlockHeadersByStartTimeStamp")
	defer span.End()
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

func (j *JSONRPCServer) GetBlockTransactions(req *http.Request, args *types.GetBlockTransactionsArgs, reply *types.SEQTransactionResponse) error {
	_, span := j.c.Tracer().Start(req.Context(), "Server.GetBlockTransactions")
	defer span.End()
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
			if action.GetTypeID() == actions.MsgID {
				ns := hex.EncodeToString(action.NMTNamespace())
				reply.Txs = append(reply.Txs, &types.SEQTransaction{
					Namespace:   ns,
					Transaction: action.(*actions.SequencerMsg).Data, // eth format tx binary
					Index:       uint64(k),
					TxID:        tx.ID().String(), // TODO: what should be the TxID for multi action tx?
				})
			}
		}
	}

	reply.BlockID = args.ID

	return nil
}

func (j *JSONRPCServer) GetCommitmentBlocks(req *http.Request, args *types.GetBlockCommitmentArgs, reply *types.SequencerWarpBlockResponse) error {
	_, span := j.c.Tracer().Start(req.Context(), "Server.GetCommitmentBlocks")
	defer span.End()
	parser := j.ServerParser(req.Context(), j.c.NetworkID(), j.c.ChainID())
	warpResp, err := j.c.Archiver().GetCommitmentBlocks(args, parser)
	if err != nil {
		return err
	}
	reply.Blocks = warpResp.Blocks

	return nil
}

func (j *JSONRPCServer) GetBlockTransactionsByNamespace(req *http.Request, args *types.GetBlockTransactionsByNamespaceArgs, reply *types.SEQTransactionResponse) error {
	_, span := j.c.Tracer().Start(req.Context(), "Server.GetBlockTransactionsByNamespace")
	defer span.End()
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
			if action.GetTypeID() == actions.MsgID {
				ns := hex.EncodeToString(action.NMTNamespace())
				if args.Namespace != ns {
					continue
				}
				reply.Txs = append(reply.Txs, &types.SEQTransaction{
					Namespace:   ns,
					Transaction: action.(*actions.SequencerMsg).Data, // eth format tx binary
					Index:       uint64(k),                           // might be duplicate
					TxID:        tx.ID().String(),
				})
			}
		}
	}

	reply.BlockID = blkID.String()

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

func (j *JSONRPCServer) ServerParser(_ context.Context, networkID uint32, chainID ids.ID) chain.Parser {
	g := j.c.Genesis()

	// The only thing this is using is the ActionRegistry and AuthRegistry so this should be fine
	return &ServerParser{networkID, chainID, g}
}

func (j *JSONRPCServer) GetAcceptedBlockWindow(req *http.Request, _ *struct{}, reply *int) error {
	_, span := j.c.Tracer().Start(req.Context(), "Server.GetAcceptedBlockWindow")
	defer span.End()
	*reply = j.c.GetAcceptedBlockWindow()
	return nil
}
