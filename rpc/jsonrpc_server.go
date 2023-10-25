// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/AnomalyFi/hypersdk/chain"
	seqconsts "github.com/AnomalyFi/nodekit-seq/consts"

	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/types"

	sequtils "github.com/AnomalyFi/nodekit-seq/utils"

	ethhex "github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/tidwall/btree"
)

type JSONRPCServer struct {
	c       Controller
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

type BlockInfo struct {
	BlockId   string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	L1Head    uint64 `json:"l1_head"`
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

type GetBlockTransactionsByNamespaceArgs struct {
	Height    uint64 `json:"height"`
	Namespace string `json:"namespace"`
}

func (j *JSONRPCServer) GetBlockHeadersByHeight(req *http.Request, args *GetBlockHeadersByHeightArgs, reply *BlockHeadersResponse) error {
	prevBlkId, success := j.idsByHeight.Get(args.Height - 1)

	Prev := BlockInfo{}
	if success {
		blk, found := j.headers.Get(prevBlkId.String())
		if !found {
			return errors.New("Could not find Block")
		}
		l1Head, err := ethhex.DecodeBig(blk.L1Head)
		if err != nil {
			return err
		}
		Prev = BlockInfo{
			BlockId:   prevBlkId.String(),
			Timestamp: blk.Tmstmp,
			L1Head:    l1Head.Uint64(),
		}
	}

	blocks := make([]BlockInfo, 0)

	Next := BlockInfo{}

	j.idsByHeight.Ascend(args.Height, func(heightKey uint64, id ids.ID) bool {
		// Does heightKey match the given block's height for the id
		blk, found := j.headers.Get(id.String())
		if !found {
			return false
		}

		l1Head, err := ethhex.DecodeBig(blk.L1Head)
		if err != nil {
			return false
		}
		if blk.Hght == heightKey {
			blocks = append(blocks, BlockInfo{
				BlockId:   id.String(),
				Timestamp: blk.Tmstmp,
				L1Head:    l1Head.Uint64(),
			})
		}

		// if blk.Hght == heightKey && blk.Tmstmp < endNumber {
		// 	blocks = append(blocks, BlockInfo{
		// 		BlockId: id,
		// 	})
		// }

		// endNumber
		// TODO do I want this as a timestamp

		if blk.Tmstmp > args.End {
			Next = BlockInfo{
				BlockId:   id.String(),
				Timestamp: blk.Tmstmp,
				L1Head:    l1Head.Uint64(),
			}
			return false
		}
		return true
	})

	reply.From = args.Height
	reply.Blocks = blocks
	reply.Prev = Prev
	reply.Next = Next

	return nil
}

func (j *JSONRPCServer) GetBlockHeadersID(req *http.Request, args *GetBlockHeadersIDArgs, reply *BlockHeadersResponse) error {
	// Parse query parameters

	var firstBlock uint64
	//var prevBlkId ids.ID

	if args.ID != "" {
		id, err := ids.FromString(args.ID)
		if err != nil {
			return err
		}
		// TODO make this into the response
		block, found := j.headers.Get(id.String())
		if !found {
			return errors.New("Could not find Block")
		}

		firstBlock = block.Hght
		// Handle hash parameter
		// ...
	} else {
		firstBlock = 1
		// Handle error or default case
		// TODO add error potentially
		// http.Error(writer, "Invalid parameters", http.StatusBadRequest)
		//TODO used to return nil but changed
		//return nil
	}

	//prevBlkId, success := j.idsByHeight.Get(firstBlock - 1)

	Prev := BlockInfo{}
	if firstBlock > 0 {
		// j.idsByHeight.Descend(firstBlock, func(heightKey uint64, id ids.ID) bool {

		// 	prevBlkId = id
		// 	return false
		// })
		prevBlkId, success := j.idsByHeight.Get(firstBlock - 1)

		if success {
			blk, found := j.headers.Get(prevBlkId.String())
			if !found {
				return errors.New("Could not find Previous Block")
			}
			l1Head, err := ethhex.DecodeBig(blk.L1Head)
			if err != nil {
				return err
			}
			Prev = BlockInfo{
				BlockId:   prevBlkId.String(),
				Timestamp: blk.Tmstmp,
				L1Head:    l1Head.Uint64(),
			}
		} else {
			return errors.New("Could not find Previous Block")
		}
	}

	blocks := make([]BlockInfo, 0)

	Next := BlockInfo{}

	j.idsByHeight.Ascend(firstBlock, func(heightKey uint64, id ids.ID) bool {
		// Does heightKey match the given block's height for the id
		blk, found := j.headers.Get(id.String())
		if !found {
			return false
		}
		l1Head, err := ethhex.DecodeBig(blk.L1Head)
		if err != nil {
			// TODO add error potentially
			return false
		}

		if blk.Hght == heightKey {
			blocks = append(blocks, BlockInfo{
				BlockId:   id.String(),
				Timestamp: blk.Tmstmp,
				L1Head:    l1Head.Uint64(),
			})
		}

		if blk.Tmstmp > args.End {
			Next = BlockInfo{
				BlockId:   id.String(),
				Timestamp: blk.Tmstmp,
				L1Head:    l1Head.Uint64(),
			}
			return false
		}
		return true
	})

	// res := BlockHeadersResponse{From: firstBlock, Blocks: blocks, Prev: Prev, Next: Next}

	// reply = &res
	reply.From = firstBlock
	reply.Blocks = blocks
	reply.Prev = Prev
	reply.Next = Next
	// TODO add blocks to the list of blocks contained in this time window
	// Marshal res to JSON and send the response

	return nil
}

func (j *JSONRPCServer) GetBlockHeadersByStart(req *http.Request, args *GetBlockHeadersByStartArgs, reply *BlockHeadersResponse) error {
	// Parse query parameters

	var firstBlock uint64

	//var prevBlkId ids.ID

	//TODO This breaks when the Start is not exactly equal to one of our blocks
	//heightFound, success := j.blocks.Get(args.Start)

	// t, h, found := j.blocks.GetAt(0)

	// if found {
	// 	err := fmt.Errorf("timestamp %d height %d.", t, h)

	// 	return err
	// }
	j.blocks.Ascend(args.Start, func(tmstp int64, height uint64) bool {

		firstBlock = height
		return false

	})

	// if success {
	// 	firstBlock = heightFound
	// }

	Prev := BlockInfo{}
	if firstBlock > 0 {
		// j.idsByHeight.Descend(firstBlock - 1, func(heightKey uint64, id ids.ID) bool {

		// 	prevBlkId = id
		// 	return false
		// })
		prevBlkId, success := j.idsByHeight.Get(firstBlock - 1)

		if success {
			blk, found := j.headers.Get(prevBlkId.String())
			if !found {
				return fmt.Errorf("Could not find Previous Block: %d ", firstBlock)
			}
			l1Head, err := ethhex.DecodeBig(blk.L1Head)
			if err != nil {
				return err
			}
			tmp := blk.Tmstmp / 1000
			Prev = BlockInfo{
				BlockId:   prevBlkId.String(),
				Timestamp: tmp,
				L1Head:    l1Head.Uint64(),
			}
		} else {
			return fmt.Errorf("Could not find Previous Block: %d, idsByHeight height %d, blocks height %d ", firstBlock, j.idsByHeight.Len(), j.blocks.Len())
		}
	} else {
		firstBlock = 1
	}

	blocks := make([]BlockInfo, 0)

	Next := BlockInfo{}

	fmt.Println("First Block:", firstBlock, "idsByHeight height:", j.idsByHeight.Len(), "blocks height:", j.blocks.Len())

	j.idsByHeight.Ascend(firstBlock, func(heightKey uint64, id ids.ID) bool {
		// Does heightKey match the given block's height for the id
		blk, found := j.headers.Get(id.String())
		fmt.Println("Block SEQ: ", blk.StateRoot.String(), "Block L1:", blk.L1Head)

		if !found {
			return false
		}
		l1Head, err := ethhex.DecodeBig(blk.L1Head)
		if err != nil {
			// TODO add error potentially
			return false
		}

		if blk.Tmstmp >= args.End {
			fmt.Println("Next Found")
			fmt.Println("Blocks Length After Next:", len(blocks))

			tmp := blk.Tmstmp / 1000

			Next = BlockInfo{
				BlockId:   id.String(),
				Timestamp: tmp,
				L1Head:    l1Head.Uint64(),
			}
			return false
		}

		if blk.Hght == heightKey {
			tmp := blk.Tmstmp / 1000

			blocks = append(blocks, BlockInfo{
				BlockId:   id.String(),
				Timestamp: tmp,
				L1Head:    l1Head.Uint64(),
			})
			fmt.Println("Match Found")
			fmt.Println("Blocks Length After Append:", len(blocks))

		}

		return true
	})

	fmt.Println("Blocks Length:", len(blocks))

	// res := BlockHeadersResponse{From: firstBlock, Blocks: blocks, Prev: Prev, Next: Next}

	// TODO add blocks to the list of blocks contained in this time window
	// Marshal res to JSON and send the response

	reply.From = firstBlock
	reply.Blocks = blocks
	reply.Prev = Prev
	reply.Next = Next

	return nil
}

func (j *JSONRPCServer) GetBlockTransactions(req *http.Request, args *GetBlockTransactionsArgs, reply *TransactionResponse) error {
	// Parse query parameters

	// TODO either the firstBlock height is equal to height or use the hash to get it or if none of the above work then use the btree to get it
	if args.ID != "" {
		return nil
	}

	// id, err := ids.FromString(args.ID)
	// if err != nil {
	// 	return err
	// }

	block, success := j.headers.Get(args.ID)

	if !success {
		return errors.New("Txs Not Found")
	}

	reply.Txs = block.Txs
	reply.BlockId = args.ID

	return nil
}

func (j *JSONRPCServer) GetBlockTransactionsByNamespace(req *http.Request, args *GetBlockTransactionsByNamespaceArgs, reply *SEQTransactionResponse) error {
	BlkId, success := j.idsByHeight.Get(args.Height)

	if !success {
		return errors.New("Txs Not Found For Namespace")
	}

	block, found := j.blocksWithValidTxs.Get(BlkId.String())
	if !found {
		return fmt.Errorf("Could not find Block: %s ", BlkId.String())
	}

	txs := block.Txs[args.Namespace]

	reply.Txs = txs
	reply.BlockId = BlkId.String()

	return nil
}

func (j *JSONRPCServer) AcceptBlock(blk *chain.StatelessBlock) error {
	ctx := context.Background()
	msg, err := rpc.PackBlockMessage(blk)
	if err != nil {
		return err
	}
	parser := j.ServerParser(ctx, 1, ids.Empty)

	block, results, _, id, err := rpc.UnpackBlockMessage(msg, parser)

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
