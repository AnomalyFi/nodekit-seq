package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto/bls"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/fees"
	hrpc "github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/ava-labs/avalanchego/ids"
)

// MultiJSONRPCClient holds SEQ and HyperSDK json rpc clients together along with authFactory.
type MultiJSONRPCClient struct {
	SeqCli  *JSONRPCClient
	HCli    *hrpc.JSONRPCClient
	AuthFac chain.AuthFactory
}

// NewMultiJSONRPCClientWithED25519Factory creates a new MultiJsonRPCClient object with ED25519 auth factory.
func NewMultiJSONRPCClientWithED25519Factory(uri string, networkID uint32, chainID ids.ID, privBytes []byte) *MultiJSONRPCClient {
	priv := ed25519.PrivateKey(privBytes)
	factory := auth.NewED25519Factory(priv)
	return &MultiJSONRPCClient{
		SeqCli:  NewJSONRPCClient(uri, networkID, chainID),
		HCli:    hrpc.NewJSONRPCClient(uri),
		AuthFac: factory,
	}
}

// NewMultiJSONRPCClientWithBLSFactory creates a new MultiJsonRPCClient object with BLS auth factory.
func NewMultiJSONRPCClientWithBLSFactory(uri string, networkID uint32, chainID ids.ID, privBytes []byte) *MultiJSONRPCClient {
	priv, _ := bls.PrivateKeyFromBytes(privBytes)
	factory := auth.NewBLSFactory(priv)
	return &MultiJSONRPCClient{
		SeqCli:  NewJSONRPCClient(uri, networkID, chainID),
		HCli:    hrpc.NewJSONRPCClient(uri),
		AuthFac: factory,
	}
}

// GenerateAndSubmitTx generates and submits a transaction with the given actions and priority fee.
func (multi *MultiJSONRPCClient) GenerateAndSubmitTx(ctx context.Context, actions []chain.Action, priorityFee uint64) (ids.ID, error) {
	parser, err := multi.SeqCli.Parser(ctx)
	if err != nil {
		return ids.Empty, err
	}

	unitPrices, err := multi.HCli.UnitPrices(ctx, true)
	if err != nil {
		return ids.Empty, err
	}

	units, feeMarketUnits, err := chain.EstimateUnits(parser.Rules(time.Now().UnixMilli()), actions, multi.AuthFac)
	if err != nil {
		return ids.Empty, err
	}

	fee, err := fees.MulSum(unitPrices, units)
	if err != nil {
		return ids.Empty, err
	}

	nss := make([]string, 0)
	for ns := range feeMarketUnits {
		nss = append(nss, ns)
	}

	nsPrices, err := multi.HCli.NameSpacesPrice(ctx, nss)
	if err != nil {
		return ids.Empty, err
	}

	for i, ns := range nss {
		fee += nsPrices[i] * feeMarketUnits[ns]
	}

	// add priority fee(if any) into the max Fee.
	fee += priorityFee
	// set max fee 20% higher than the pessimistic estimation.
	fee += (fee / 5)

	now := time.Now().UnixMilli()
	rules := parser.Rules(now)

	base := &chain.Base{
		Timestamp: utils.UnixRMilli(now, rules.GetValidityWindow()),
		ChainID:   multi.SeqCli.chainID,
		MaxFee:    fee,
	}

	// Build transaction
	actionRegistry, authRegistry := parser.Registry()
	tx := chain.NewTx(base, actions)
	tx, err = tx.Sign(multi.AuthFac, actionRegistry, authRegistry)
	if err != nil {
		return ids.Empty, fmt.Errorf("%w: failed to sign transaction", err)
	}
	return multi.HCli.SubmitTx(ctx, tx.Bytes())
}
