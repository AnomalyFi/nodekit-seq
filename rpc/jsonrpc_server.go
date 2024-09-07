// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/fees"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/auth"
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

type SubmitMsgTxArgs struct {
	ChainID          string   `json:"chain_id"`
	NetworkID        uint32   `json:"network_id"`
	SecondaryChainID []byte   `json:"secondary_chain_id"`
	Data             [][]byte `json:"data"`
}

type SubmitMsgTxReply struct {
	TxID string `json:"txId"`
}

// This method submits SequencerMsg actions to the chain, without any priority fee.
// Note: This method will be removed in the future, as it is only used for testing.
func (j *JSONRPCServer) SubmitMsgTx(
	req *http.Request,
	args *SubmitMsgTxArgs,
	reply *SubmitMsgTxReply,
) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.SubmitMsgTx")
	defer span.End()

	chainID, err := ids.FromString(args.ChainID)
	if err != nil {
		return err
	}

	unitPrices, err := j.c.UnitPrices(ctx)
	if err != nil {
		return err
	}

	parser := j.ServerParser(ctx, args.NetworkID, chainID)

	privBytes, err := codec.LoadHex(
		"323b1d8f4eed5f0da9da93071b034f2dce9d2d22692c172f3cb252a64ddfafd01b057de320297c29ad0c1f589ea216869cf1938d88c9fbd70d6748323dbf2fa7", //nolint:lll
		ed25519.PrivateKeyLen,
	)
	if err != nil {
		return err
	}

	priv := ed25519.PrivateKey(privBytes)
	factory := auth.NewED25519Factory(priv)

	tpriv, err := ed25519.GeneratePrivateKey()
	if err != nil {
		return err
	}

	rsender := auth.NewED25519Address(tpriv.PublicKey())

	acts := make([]chain.Action, 0, len(args.Data))
	for _, data := range args.Data {
		act := actions.SequencerMsg{
			FromAddress: rsender,
			Data:        data,
			ChainID:     args.SecondaryChainID,
			RelayerID:   0,
		}
		acts = append(acts, &act)
	}

	maxUnits, feeMarketUnits, err := chain.EstimateUnits(parser.Rules(time.Now().UnixMilli()), acts, factory)
	if err != nil {
		return err
	}
	maxFee, err := fees.MulSum(unitPrices, maxUnits)
	if err != nil {
		return err
	}

	nss := make([]string, 0)
	for ns := range feeMarketUnits {
		nss = append(nss, ns)
	}

	nsPrices, err := j.c.NameSpacesPrice(ctx, nss)
	if err != nil {
		return err
	}

	for i, ns := range nss {
		maxFee += nsPrices[i] * feeMarketUnits[ns]
	}

	// Add 20% to the max fee
	maxFee += (maxFee / 5)

	now := time.Now().UnixMilli()
	rules := parser.Rules(now)

	base := &chain.Base{
		Timestamp: utils.UnixRMilli(now, rules.GetValidityWindow()),
		ChainID:   chainID,
		MaxFee:    maxFee,
	}

	// Build transaction
	actionRegistry, authRegistry := parser.Registry()
	tx := chain.NewTx(base, acts)
	tx, err = tx.Sign(factory, actionRegistry, authRegistry)
	if err != nil {
		return fmt.Errorf("%w: failed to sign transaction", err)
	}

	msg, err := tx.Digest()
	if err != nil {
		// Should never occur because populated during unmarshal
		return err
	}
	if err := tx.Auth.Verify(ctx, msg); err != nil {
		return err
	}
	txID := tx.ID()
	reply.TxID = txID.String()
	return j.c.Submit(ctx, false, []*chain.Transaction{tx})[0]
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
