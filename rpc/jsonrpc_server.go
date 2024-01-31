// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/nodekit-seq/archiver"
	seqconsts "github.com/AnomalyFi/nodekit-seq/consts"

	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/types"

	sequtils "github.com/AnomalyFi/nodekit-seq/utils"

	"github.com/tidwall/btree"
)

type JSONRPCServer struct {
	c Controller

	headers *types.ShardedMap[string, *chain.StatefulBlock]
	//map[ids.ID]*chain.StatefulBlock // Map block ID to block header

	blocksWithValidTxs *types.ShardedMap[string, *types.SequencerBlock]

	//map[ids.ID]*types.SequencerBlock // Map block ID to block header

	idsByHeight btree.Map[uint64, ids.ID] // Map block ID to block height

	// tmstp, height
	blocks btree.Map[int64, uint64]
}

func NewJSONRPCServer(c Controller) *JSONRPCServer {
	headers := types.NewShardedMap[string, *chain.StatefulBlock](10000, 10, types.HashString)
	blocksWithValidTxs := types.NewShardedMap[string, *types.SequencerBlock](10000, 10, types.HashString)

	return &JSONRPCServer{
		c:                  c,
		headers:            headers,
		blocksWithValidTxs: blocksWithValidTxs,
		idsByHeight:        btree.Map[uint64, ids.ID]{},
		blocks:             btree.Map[int64, uint64]{},
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
	ChainId          string `json:"chain_id"`
	NetworkID        uint32 `json:"network_id"`
	SecondaryChainId []byte `json:"secondary_chain_id"`
	Data             []byte `json:"data"`
}

type SubmitMsgTxReply struct {
	TxID string `json:"txId"`
}

func (j *JSONRPCServer) SubmitMsgTx(
	req *http.Request,
	args *SubmitMsgTxArgs,
	reply *SubmitMsgTxReply,
) error {
	ctx := context.Background()

	chainId, err := ids.FromString(args.ChainId)
	if err != nil {
		return err
	}

	unitPrices, err := j.c.UnitPrices(ctx)
	if err != nil {
		return err
	}

	parser := j.ServerParser(ctx, args.NetworkID, chainId)

	priv, err := ed25519.HexToKey(
		"323b1d8f4eed5f0da9da93071b034f2dce9d2d22692c172f3cb252a64ddfafd01b057de320297c29ad0c1f589ea216869cf1938d88c9fbd70d6748323dbf2fa7", //nolint:lll
	)
	factory := auth.NewED25519Factory(priv)

	tpriv, err := ed25519.GeneratePrivateKey()
	if err != nil {
		return err
	}

	trsender := tpriv.PublicKey()
	action := &actions.SequencerMsg{
		FromAddress: trsender,
		Data:        args.Data,
		ChainId:     args.SecondaryChainId,
	}
	// TODO need to define action, authFactory
	maxUnits, err := chain.EstimateMaxUnits(parser.Rules(time.Now().UnixMilli()), action, factory, nil)
	if err != nil {
		return err
	}
	maxFee, err := chain.MulSum(unitPrices, maxUnits)
	if err != nil {
		return err
	}

	// TODO above is generateTransaction below is generateTransactionManual

	now := time.Now().UnixMilli()
	rules := parser.Rules(now)

	base := &chain.Base{
		Timestamp: utils.UnixRMilli(now, rules.GetValidityWindow()),
		ChainID:   chainId,
		MaxFee:    maxFee,
	}

	// Build transaction
	actionRegistry, authRegistry := parser.Registry()
	tx := chain.NewTx(base, nil, action, false)
	tx, err = tx.Sign(factory, actionRegistry, authRegistry)
	if err != nil {
		return fmt.Errorf("%w: failed to sign transaction", err)
	}

	// TODO above is new!

	if err := tx.AuthAsyncVerify()(); err != nil {
		return err
	}
	txID := tx.ID()
	reply.TxID = txID.String()
	return j.c.Submit(ctx, false, []*chain.Transaction{tx})[0]
}

type account struct {
	priv    ed25519.PrivateKey
	factory *auth.ED25519Factory
	rsender ed25519.PublicKey
	sender  string
}

type TxArgs struct {
	TxID ids.ID `json:"txId"`
}

type TxReply struct {
	Timestamp int64            `json:"timestamp"`
	Success   bool             `json:"success"`
	Units     chain.Dimensions `json:"units"`
	Fee       uint64           `json:"fee"`
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
	Warp     bool   `json:"warp"`
}

func (j *JSONRPCServer) Asset(req *http.Request, args *AssetArgs, reply *AssetReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Asset")
	defer span.End()

	exists, symbol, decimals, metadata, supply, owner, warp, err := j.c.GetAssetFromState(ctx, args.Asset)
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
	reply.Owner = sequtils.Address(owner)
	reply.Warp = warp
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

	addr, err := sequtils.ParseAddress(args.Address)
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

type LoanArgs struct {
	Destination ids.ID `json:"destination"`
	Asset       ids.ID `json:"asset"`
}

type LoanReply struct {
	Amount uint64 `json:"amount"`
}

func (j *JSONRPCServer) Loan(req *http.Request, args *LoanArgs, reply *LoanReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Loan")
	defer span.End()

	amount, err := j.c.GetLoanFromState(ctx, args.Asset, args.Destination)
	if err != nil {
		return err
	}
	reply.Amount = amount
	return nil
}

// TODO need to fix this. Tech debt
type TransactionResponse struct {
	Txs     []*chain.Transaction `json:"txs"`
	BlockId string               `json:"id"`
}

type SEQTransactionResponse struct {
	Txs       []*types.SEQTransaction `json:"txs"`
	BlockId   string                  `json:"id"`
	Timestamp int64                   `json:"timestamp"`
	L1Head    uint64                  `json:"l1_head"`
	Height    uint64                  `json:"height"`
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
	return j.c.GetByHeight(args.Height, args.End, reply)
}

func (j *JSONRPCServer) GetBlockHeadersID(req *http.Request, args *types.GetBlockHeadersIDArgs, reply *types.BlockHeadersResponse) error {
	return j.c.GetByID(args, reply)
}

func (j *JSONRPCServer) GetBlockHeadersByStart(req *http.Request, args *types.GetBlockHeadersByStartArgs, reply *types.BlockHeadersResponse) error {
	return j.c.GetByStart(args, reply)
}

func (j *JSONRPCServer) GetBlockTransactions(req *http.Request, args *GetBlockTransactionsArgs, reply *TransactionResponse) error {
	if args.ID != "" {
		return nil
	}

	ctx, span := j.c.Tracer().Start(req.Context(), "Server.GetBlockTransactions")
	defer span.End()

	dbBlock := archiver.DBBlock{
		BlockId: args.ID,
	}

	block, id, err := j.c.GetBlockFromArchiver(ctx, &dbBlock)
	if err != nil {
		return err
	}

	reply.Txs = block.Txs
	reply.BlockId = id.String()

	return nil
}

func (j *JSONRPCServer) GetCommitmentBlocks(req *http.Request, args *GetBlockCommitmentArgs, reply *SequencerWarpBlockResponse) error {
	// Parse query parameters
	if args.First < 1 {
		return nil
	}

	blocks := make([]SequencerWarpBlock, 0)

	j.idsByHeight.Ascend(args.First, func(heightKey uint64, id ids.ID) bool {
		// Does heightKey match the given block's height for the id
		if len(blocks) >= args.MaxBlocks {
			return false
		}

		blockTemp, success := j.headers.Get(id.String())
		if !success {
			return success
		}

		header := &types.Header{
			Height:    blockTemp.Hght,
			Timestamp: uint64(blockTemp.Tmstmp),
			L1Head:    uint64(blockTemp.L1Head),
			TransactionsRoot: types.NmtRoot{
				Root: id[:],
			},
		}

		comm := header.Commit()

		//TODO swapped these 2 functions so now it exits earlier. Need to test
		if blockTemp.Hght >= args.CurrentHeight {
			parentRoot := types.NewU256().SetBytes(blockTemp.Prnt)
			bigParentRoot := parentRoot.Int

			blocks = append(blocks, SequencerWarpBlock{
				BlockId:    id.String(),
				Timestamp:  blockTemp.Tmstmp,
				L1Head:     uint64(blockTemp.L1Head),
				Height:     big.NewInt(int64(blockTemp.Hght)),
				BlockRoot:  &comm.Uint256().Int,
				ParentRoot: &bigParentRoot,
			})
			return false
		}

		if blockTemp.Hght == heightKey {
			parentRoot := types.NewU256().SetBytes(blockTemp.Prnt)
			bigParentRoot := parentRoot.Int

			blocks = append(blocks, SequencerWarpBlock{
				BlockId:    id.String(),
				Timestamp:  blockTemp.Tmstmp,
				L1Head:     uint64(blockTemp.L1Head),
				Height:     big.NewInt(int64(blockTemp.Hght)),
				BlockRoot:  &comm.Uint256().Int,
				ParentRoot: &bigParentRoot,
			})
		}

		return true
	})

	reply.Blocks = blocks

	return nil
}

func (j *JSONRPCServer) GetBlockTransactionsByNamespace(req *http.Request, args *GetBlockTransactionsByNamespaceArgs, reply *SEQTransactionResponse) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.GetBlockTransactionsByNamespace")
	defer span.End()

	dbBlock := archiver.DBBlock{
		Height: args.Height,
	}

	blk, id, err := j.c.GetBlockFromArchiver(ctx, &dbBlock)
	if err != nil {
		return err
	}

	seq_txs := make([]*types.SEQTransaction, 0)

	for i, tx := range blk.Txs {
		if contains(blk.NMTNamespaceToTxIndexes[args.Namespace], i) {
			switch action := tx.Action.(type) {
			case *actions.SequencerMsg:
				hx := hex.EncodeToString(action.ChainId)
				new_tx := types.SEQTransaction{
					Namespace:   hx,
					Tx_id:       tx.ID().String(),
					Transaction: action.Data,
					Index:       uint64(i),
				}
				seq_txs = append(seq_txs, &new_tx)
			}
		}

	}

	reply.Height = blk.Hght
	reply.L1Head = uint64(blk.L1Head)
	reply.Timestamp = blk.Tmstmp
	reply.Txs = seq_txs
	reply.BlockId = id.String()

	//TODO add on the NMT Proof

	return nil
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (j *JSONRPCServer) AcceptBlock(blk *chain.StatelessBlock) error {
	ctx := context.Background()
	msg, err := rpc.PackBlockMessage(blk)
	if err != nil {
		return err
	}
	parser := j.ServerParser(ctx, 1, ids.Empty)

	block, results, _, id, err := rpc.UnpackBlockMessage(msg, parser)

	//TODO I should experiment with TTL
	j.headers.Put(id.String(), block)
	// j.headers[id.String()] = block
	j.idsByHeight.Set(block.Hght, *id)
	j.blocks.Set(block.Tmstmp, block.Hght)

	// TODO I need to call CommitmentManager.AcceptBlock here because otherwise the unpacking will be a pain

	seq_txs := make(map[string][]*types.SEQTransaction)

	for i, tx := range blk.Txs {
		result := results[i]

		if result.Success {
			switch action := tx.Action.(type) {
			case *actions.SequencerMsg:
				hx := hex.EncodeToString(action.ChainId)
				if seq_txs[hx] == nil {
					seq_txs[hx] = make([]*types.SEQTransaction, 0)
				}
				new_tx := types.SEQTransaction{
					Namespace:   hx,
					Tx_id:       tx.ID().String(),
					Transaction: action.Data,
					Index:       uint64(i),
				}
				seq_txs[hx] = append(seq_txs[hx], &new_tx)
			}
		}

	}

	sequencerBlock := &types.SequencerBlock{
		StateRoot: blk.StateRoot,
		Prnt:      blk.Prnt,
		Tmstmp:    blk.Tmstmp,
		Hght:      blk.Hght,
		Txs:       seq_txs,
	}

	// TODO need to fix this
	j.blocksWithValidTxs.Put(id.String(), sequencerBlock)

	return nil
}

type BlockArgs struct {
	// either of them need to be fed
	ID     ids.ID `json:"id"`
	Height uint64 `json:"height"`
}

type BlockReply struct {
	Block []byte `json:"block"`
}

func (j *JSONRPCServer) Block(req *http.Request, args *BlockArgs, reply *BlockReply) error {
	ctx, span := j.c.Tracer().Start(req.Context(), "Server.Block")
	defer span.End()

	var BlockID string
	// reason of doing this is that Gorm only treat 0, nil, "" as null values
	// ids.Empty.String() won't return a null string
	// see: https://gorm.io/docs/query.html#Struct-amp-Map-Conditions
	if !bytes.Equal(args.ID[:], ids.Empty[:]) {
		BlockID = args.ID.String()
	} else {
		BlockID = ""
	}

	dbBlock := archiver.DBBlock{
		BlockId: BlockID,
		Height:  args.Height,
	}
	blk, _, err := j.c.GetBlockFromArchiver(ctx, &dbBlock)
	if err != nil {
		return err
	}

	reply.Block, err = blk.Marshal()
	if err != nil {
		return err
	}

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
