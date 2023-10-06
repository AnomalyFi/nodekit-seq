// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"encoding/hex"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/utils"

	"github.com/tidwall/btree"
)

type JSONRPCServer struct {
	c       Controller
	headers map[ids.ID]*chain.StatefulBlock // Map block ID to block header

	blocksWithValidTxs map[ids.ID]*SequencerBlock // Map block ID to block header

	idsByHeight btree.Map[uint64, ids.ID] // Map block ID to block height

	//tmstp, height
	blocks btree.Map[int64, uint64]
}

func NewJSONRPCServer(c Controller) *JSONRPCServer {
	return &JSONRPCServer{
		c:                  c,
		headers:            map[ids.ID]*chain.StatefulBlock{},
		blocksWithValidTxs: map[ids.ID]*SequencerBlock{},
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
	reply.Owner = utils.Address(owner)
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

	addr, err := utils.ParseAddress(args.Address)
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
	BlockId ids.ID `json:"id"`
}

type BlockHeadersResponse struct {
	From   uint64      `json:"from"`
	Blocks []BlockInfo `json:"blocks"`
	Prev   BlockInfo   `json:"prev"`
	Next   BlockInfo   `json:"next"`
}

type TransactionResponse struct {
	Txs     []*chain.Transaction `json:"txs"`
	BlockId ids.ID               `json:"id"`
}

// TODO need to fix this. Tech debt
type SEQTransactionResponse struct {
	Txs     []*SEQTransaction `json:"txs"`
	BlockId ids.ID            `json:"id"`
}

type SEQTransaction struct {
	Namespace   string `json:"namespace"`
	Tx_id       ids.ID `json:"tx_id"`
	Index       uint64 `json:"tx_index"`
	Transaction []byte `json:"transaction"`
}

type SequencerBlock struct {
	StateRoot ids.ID                       `json:"state_root"`
	Prnt      ids.ID                       `json:"parent"`
	Tmstmp    int64                        `json:"timestamp"`
	Hght      uint64                       `json:"height"`
	Txs       map[string][]*SEQTransaction `json:"transactions"`
}

type GetBlockHeadersByHeightArgs struct {
	Height uint64 `json:"height"`
	End    int64  `json:"end"`
}

type GetBlockHeadersIDArgs struct {
	ID  ids.ID `json:"id"`
	End int64  `json:"end"`
}

type GetBlockHeadersByStartArgs struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

type GetBlockTransactionsArgs struct {
	ID ids.ID `json:"block_id"`
}

type GetBlockTransactionsByNamespaceArgs struct {
	ID        ids.ID `json:"block_id"`
	Namespace string `json:"namespace"`
}

func (j *JSONRPCServer) getBlockHeadersByHeight(req *http.Request, args *GetBlockHeadersByHeightArgs, reply *BlockHeadersResponse) error {

	prevBlkId, success := j.idsByHeight.Get(args.Height - 1)

	Prev := BlockInfo{}
	if success {
		Prev = BlockInfo{
			BlockId: prevBlkId,
		}
	}

	blocks := make([]BlockInfo, 0)

	Next := BlockInfo{}

	j.idsByHeight.Ascend(args.Height, func(heightKey uint64, id ids.ID) bool {
		//Does heightKey match the given block's height for the id
		blk := j.headers[id]

		if blk.Hght == heightKey {
			blocks = append(blocks, BlockInfo{
				BlockId: id,
			})
		}

		// if blk.Hght == heightKey && blk.Tmstmp < endNumber {
		// 	blocks = append(blocks, BlockInfo{
		// 		BlockId: id,
		// 	})
		// }

		//endNumber
		//TODO do I want this as a timestamp

		if blk.Tmstmp > args.End {
			Next = BlockInfo{
				BlockId: id,
			}
			return false
		}
		return true

	})

	res := BlockHeadersResponse{From: args.Height, Blocks: blocks, Prev: Prev, Next: Next}

	reply = &res

	return nil
}

func (j *JSONRPCServer) getBlockHeadersID(req *http.Request, args *GetBlockHeadersIDArgs, reply *BlockHeadersResponse) error {
	// Parse query parameters

	var firstBlock uint64

	if args.ID != ids.Empty {

		//TODO make this into the response
		block := j.headers[args.ID]

		firstBlock = block.Hght
		// Handle hash parameter
		// ...
	} else {
		firstBlock = 0
		// Handle error or default case
		//TODO add error potentially
		// http.Error(writer, "Invalid parameters", http.StatusBadRequest)
		return nil
	}

	prevBlkId, success := j.idsByHeight.Get(firstBlock - 1)

	Prev := BlockInfo{}
	if success {
		Prev = BlockInfo{
			BlockId: prevBlkId,
		}
	}

	blocks := make([]BlockInfo, 0)

	Next := BlockInfo{}

	j.idsByHeight.Ascend(firstBlock, func(heightKey uint64, id ids.ID) bool {
		//Does heightKey match the given block's height for the id
		blk := j.headers[id]

		if blk.Hght == heightKey {
			blocks = append(blocks, BlockInfo{
				BlockId: id,
			})
		}

		if blk.Tmstmp > args.End {
			Next = BlockInfo{
				BlockId: id,
			}
			return false
		}
		return true

	})

	res := BlockHeadersResponse{From: firstBlock, Blocks: blocks, Prev: Prev, Next: Next}

	reply = &res
	//TODO add blocks to the list of blocks contained in this time window
	// Marshal res to JSON and send the response

	return nil

}

func (j *JSONRPCServer) getBlockHeadersByStart(req *http.Request, args *GetBlockHeadersByStartArgs, reply *BlockHeadersResponse) error {
	// Parse query parameters

	var firstBlock uint64

	//TODO either the firstBlock height is equal to height or use the hash to get it or if none of the above work then use the btree to get it
	heightFound, success := j.blocks.Get(args.Start)

	if success {
		firstBlock = heightFound
	}

	prevBlkId, success := j.idsByHeight.Get(firstBlock - 1)

	Prev := BlockInfo{}
	if success {
		Prev = BlockInfo{
			BlockId: prevBlkId,
		}
	}

	blocks := make([]BlockInfo, 0)

	Next := BlockInfo{}

	j.idsByHeight.Ascend(firstBlock, func(heightKey uint64, id ids.ID) bool {
		//Does heightKey match the given block's height for the id
		blk := j.headers[id]

		if blk.Hght == heightKey {
			blocks = append(blocks, BlockInfo{
				BlockId: id,
			})
		}

		if blk.Tmstmp > args.End {
			Next = BlockInfo{
				BlockId: id,
			}
			return false
		}
		return true

	})

	res := BlockHeadersResponse{From: firstBlock, Blocks: blocks, Prev: Prev, Next: Next}

	//TODO add blocks to the list of blocks contained in this time window
	// Marshal res to JSON and send the response

	reply = &res

	return nil

}

func (j *JSONRPCServer) getBlockTransactions(req *http.Request, args *GetBlockTransactionsArgs, reply *TransactionResponse) error {
	// Parse query parameters

	//TODO either the firstBlock height is equal to height or use the hash to get it or if none of the above work then use the btree to get it

	//TODO make this into the response
	block := j.headers[args.ID]

	res := TransactionResponse{Txs: block.Txs, BlockId: args.ID}

	reply = &res

	return nil
}

func (j *JSONRPCServer) getBlockTransactionsByNamespace(req *http.Request, args *GetBlockTransactionsByNamespaceArgs, reply *SEQTransactionResponse) error {
	block := j.blocksWithValidTxs[args.ID]

	txs := block.Txs[args.Namespace]
	res := SEQTransactionResponse{Txs: txs, BlockId: args.ID}

	reply = &res

	return nil
}

func (j *JSONRPCServer) AcceptBlock(blk *chain.StatelessBlock) error {
	//TODO this should take in this data so I can have methods to return it. I can easily pack the StatelessBlock and then just unpack it so I get the stuff I need
	ctx := context.Background()
	msg, err := rpc.PackBlockMessage(blk)

	if err != nil {
		return err
	}
	parser := j.ServerParser(ctx)

	//TODO need to add back SequencerOPBlock so I can get txs by chainId
	block, results, _, id, err := rpc.UnpackBlockMessage(msg, parser)

	j.headers[*id] = block
	j.idsByHeight.Set(block.Hght, *id)
	j.blocks.Set(block.Tmstmp, block.Hght)

	seq_txs := make(map[string][]*SEQTransaction)

	for i, tx := range blk.Txs {
		result := results[i]

		if result.Success {
			switch action := tx.Action.(type) {
			case *actions.SequencerMsg:
				hx := hex.EncodeToString(action.ChainId)
				if seq_txs[hx] == nil {
					seq_txs[hx] = make([]*SEQTransaction, 0)
				}
				new_tx := SEQTransaction{
					Namespace:   hx,
					Tx_id:       tx.ID(),
					Transaction: action.Data,
					Index:       uint64(i),
				}
				seq_txs[hx] = append(seq_txs[hx], &new_tx)
			}
		}

	}

	sequencerBlock := &SequencerBlock{
		StateRoot: blk.StateRoot,
		Prnt:      blk.Prnt,
		Tmstmp:    blk.Tmstmp,
		Hght:      blk.Hght,
		Txs:       seq_txs,
	}

	//TODO need to fix this
	j.blocksWithValidTxs[*id] = sequencerBlock

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
	return consts.ActionRegistry, consts.AuthRegistry
}

func (j *JSONRPCServer) ServerParser(ctx context.Context) chain.Parser {
	g := j.c.Genesis()

	//The only thing this is using is the ActionRegistry and AuthRegistry so this should be fine
	return &Parser{1, ids.Empty, g}
}
