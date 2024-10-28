// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"strings"

	"github.com/ava-labs/avalanchego/ids"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/requester"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	_ "github.com/AnomalyFi/nodekit-seq/registry" // ensure registry populated
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/AnomalyFi/nodekit-seq/types"
)

type JSONRPCClient struct {
	requester *requester.EndpointRequester

	networkID uint32
	chainID   ids.ID
	g         *genesis.Genesis
}

// New creates a new client object.
func NewJSONRPCClient(uri string, networkID uint32, chainID ids.ID) *JSONRPCClient {
	uri = strings.TrimSuffix(uri, "/")
	uri += JSONRPCEndpoint
	req := requester.New(uri, consts.Name)
	return &JSONRPCClient{
		requester: req,
		networkID: networkID,
		chainID:   chainID,
	}
}

func (cli *JSONRPCClient) SubmitMsgTx(ctx context.Context, chainID string, networkID uint32, secondaryChainID []byte, data [][]byte) (string, error) {
	resp := new(SubmitMsgTxReply)
	err := cli.requester.SendRequest(
		ctx,
		"submitMsgTx",
		&SubmitMsgTxArgs{
			ChainID:          chainID,
			NetworkID:        networkID,
			SecondaryChainID: secondaryChainID,
			Data:             data,
		},
		resp,
	)
	return resp.TxID, err
}

func (cli *JSONRPCClient) Genesis(ctx context.Context) (*genesis.Genesis, error) {
	if cli.g != nil {
		return cli.g, nil
	}

	resp := new(GenesisReply)
	err := cli.requester.SendRequest(
		ctx,
		"genesis",
		nil,
		resp,
	)
	if err != nil {
		return nil, err
	}
	cli.g = resp.Genesis
	return resp.Genesis, nil
}

func (cli *JSONRPCClient) Tx(ctx context.Context, id ids.ID) (bool, bool, int64, uint64, error) {
	resp := new(TxReply)
	err := cli.requester.SendRequest(
		ctx,
		"tx",
		&TxArgs{TxID: id},
		resp,
	)
	switch {
	// We use string parsing here because the JSON-RPC library we use may not
	// allows us to perform errors.Is.
	case err != nil && strings.Contains(err.Error(), ErrTxNotFound.Error()):
		return false, false, -1, 0, nil
	case err != nil:
		return false, false, -1, 0, err
	}
	return true, resp.Success, resp.Timestamp, resp.Fee, nil
}

func (cli *JSONRPCClient) Balance(ctx context.Context, addr string) (uint64, error) {
	resp := new(BalanceReply)
	err := cli.requester.SendRequest(
		ctx,
		"balance",
		&BalanceArgs{
			Address: addr,
		},
		resp,
	)
	return resp.Amount, err
}

func (cli *JSONRPCClient) GetBlockHeadersByHeight(
	ctx context.Context,
	height uint64,
	endTimeStamp int64,
) (*types.BlockHeadersResponse, error) {
	resp := new(types.BlockHeadersResponse)
	err := cli.requester.SendRequest(
		ctx,
		"getBlockHeadersByHeight",
		&types.GetBlockHeadersByHeightArgs{
			Height:       height,
			EndTimeStamp: endTimeStamp,
		},
		resp,
	)
	return resp, err
}

func (cli *JSONRPCClient) GetBlockHeadersID(
	ctx context.Context,
	id string,
	endTimeStamp int64,
) (*types.BlockHeadersResponse, error) {
	resp := new(types.BlockHeadersResponse)
	err := cli.requester.SendRequest(
		ctx,
		"getBlockHeadersID",
		&types.GetBlockHeadersIDArgs{
			ID:           id,
			EndTimeStamp: endTimeStamp,
		},
		resp,
	)
	return resp, err
}

func (cli *JSONRPCClient) GetBlockHeadersByStartTimeStamp(
	ctx context.Context,
	startTimeStamp int64,
	endTimeStamp int64,
) (*types.BlockHeadersResponse, error) {
	resp := new(types.BlockHeadersResponse)
	err := cli.requester.SendRequest(
		ctx,
		"getBlockHeadersByStartTimeStamp",
		&types.GetBlockHeadersByStartTimeStampArgs{
			StartTimeStamp: startTimeStamp,
			EndTimeStamp:   endTimeStamp,
		},
		resp,
	)
	return resp, err
}

func (cli *JSONRPCClient) GetBlockTransactions(
	ctx context.Context,
	id string,
) (*types.SEQTransactionResponse, error) {
	resp := new(types.SEQTransactionResponse)
	err := cli.requester.SendRequest(
		ctx,
		"getBlockTransactions",
		&types.GetBlockTransactionsArgs{
			ID: id,
		},
		resp,
	)
	return resp, err
}

func (cli *JSONRPCClient) GetBlockTransactionsByNamespace(
	ctx context.Context,
	height uint64,
	namespace string,
) (*types.SEQTransactionResponse, error) {
	resp := new(types.SEQTransactionResponse)
	err := cli.requester.SendRequest(
		ctx,
		"getBlockTransactionsByNamespace",
		&types.GetBlockTransactionsByNamespaceArgs{
			Height:    height,
			Namespace: namespace,
		},
		resp,
	)
	return resp, err
}

func (cli *JSONRPCClient) GetCommitmentBlocks(
	ctx context.Context,
	first uint64,
	height uint64,
	maxBlocks int,
) (*types.SequencerWarpBlockResponse, error) {
	resp := new(types.SequencerWarpBlockResponse)
	err := cli.requester.SendRequest(
		ctx,
		"getCommitmentBlocks",
		&types.GetBlockCommitmentArgs{
			First:         first,
			CurrentHeight: height,
			MaxBlocks:     maxBlocks,
		},
		resp,
	)
	return resp, err
}

func (cli *JSONRPCClient) GetAcceptedBlockWindow(ctx context.Context) (int, error) {
	resp := new(int)
	err := cli.requester.SendRequest(
		ctx,
		"getAcceptedBlockWindow",
		nil,
		resp,
	)
	return *resp, err
}

// TODO add more methods
func (cli *JSONRPCClient) WaitForBalance(
	ctx context.Context,
	addr string,
	min uint64,
) error {
	return rpc.Wait(ctx, func(ctx context.Context) (bool, error) {
		balance, err := cli.Balance(ctx, addr)
		if err != nil {
			return false, err
		}
		shouldExit := balance >= min
		if !shouldExit {
			utils.Outf(
				"{{yellow}}waiting for %s balance: %s{{/}}\n",
				utils.FormatBalance(min, consts.Decimals),
				addr,
			)
		}
		return shouldExit, nil
	})
}

func (cli *JSONRPCClient) WaitForTransaction(ctx context.Context, txID ids.ID) (bool, uint64, error) {
	var success bool
	var fee uint64
	if err := rpc.Wait(ctx, func(ctx context.Context) (bool, error) {
		found, isuccess, _, ifee, err := cli.Tx(ctx, txID)
		if err != nil {
			return false, err
		}
		fee = ifee
		success = isuccess
		return found, nil
	}); err != nil {
		return false, 0, err
	}
	return success, fee, nil
}

func (cli *JSONRPCClient) RegisteredAnchors(ctx context.Context) ([][]byte, []*hactions.AnchorInfo, error) {
	resp := new(types.RegisteredAnchorReply)
	err := cli.requester.SendRequest(
		ctx,
		"registeredAnchors",
		nil,
		resp,
	)
	return resp.Namespaces, resp.Anchors, err
}

func (cli *JSONRPCClient) GetEpochExits(ctx context.Context, epoch uint64) (*storage.EpochExitInfo, error) {
	resp := new(types.EpochExitsReply)
	err := cli.requester.SendRequest(
		ctx,
		"getEpochExits",
		epoch,
		resp,
	)
	return resp.Info, err
}

func (cli *JSONRPCClient) GetBuilder(ctx context.Context, epoch uint64) ([48]byte, error) {
	resp := new([48]byte)
	err := cli.requester.SendRequest(
		ctx,
		"getBuilder",
		epoch,
		resp,
	)

	return *resp, err
}

var _ chain.Parser = (*Parser)(nil)

type Parser struct {
	networkID uint32
	chainID   ids.ID
	genesis   *genesis.Genesis
}

func (p *Parser) ChainID() ids.ID {
	return p.chainID
}

func (p *Parser) Rules(t int64) chain.Rules {
	return p.genesis.Rules(t, p.networkID, p.chainID, []codec.Address{})
}

func (*Parser) Registry() (chain.ActionRegistry, chain.AuthRegistry) {
	return consts.ActionRegistry, consts.AuthRegistry
}

func (cli *JSONRPCClient) Parser(ctx context.Context) (chain.Parser, error) {
	g, err := cli.Genesis(ctx)
	if err != nil {
		return nil, err
	}
	return &Parser{cli.networkID, cli.chainID, g}, nil
}
