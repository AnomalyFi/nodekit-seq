// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"reflect"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/cli"
	"github.com/AnomalyFi/hypersdk/codec"

	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"

	tconsts "github.com/AnomalyFi/nodekit-seq/consts"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
	"github.com/ava-labs/avalanchego/ids"
)

// sendAndWait may not be used concurrently
func sendAndWait(
	ctx context.Context, action []chain.Action, cli *rpc.JSONRPCClient,
	scli *rpc.WebSocketClient, tcli *trpc.JSONRPCClient, factory chain.AuthFactory, priorityFee uint64, printStatus bool,
) (ids.ID, error) {
	parser, err := tcli.Parser(ctx)
	if err != nil {
		return ids.Empty, err
	}
	_, tx, _, err := cli.GenerateTransaction(ctx, parser, action, factory, priorityFee)
	if err != nil {
		return ids.Empty, err
	}

	if err := scli.RegisterTx(tx); err != nil {
		return ids.Empty, err
	}
	var res *chain.Result
	for {
		txID, dErr, result, err := scli.ListenTx(ctx)
		if dErr != nil {
			return ids.Empty, dErr
		}
		if err != nil {
			return ids.Empty, err
		}
		if txID == tx.ID() {
			res = result
			break
		}
		// TODO: don't drop these results (may be needed by a different connection)
		utils.Outf("{{yellow}}skipping unexpected transaction:{{/}} %s\n", tx.ID())
	}
	if printStatus {
		handler.Root().PrintStatus(tx.ID(), res.Success)
	}
	return tx.ID(), nil
}

func handleTx(tx *chain.Transaction, result *chain.Result) {
	actor := tx.Auth.Actor()
	if !result.Success {
		utils.Outf(
			"%s {{yellow}}%s{{/}} {{yellow}}actor:{{/}} %s {{yellow}}error:{{/}} [%s] {{yellow}}fee (max %.2f%%):{{/}} %s %s {{yellow}}consumed:{{/}} [%s]\n",
			"❌",
			tx.ID(),
			codec.MustAddressBech32(tconsts.HRP, actor),
			result.Error,
			float64(result.Fee)/float64(tx.Base.MaxFee)*100,
			utils.FormatBalance(result.Fee, tconsts.Decimals),
			tconsts.Symbol,
			cli.ParseDimensions(result.Units),
		)
		return
	}

	for _, act := range tx.Actions {
		var summaryStr string
		switch action := act.(type) {
		case *actions.Transfer:
			amountStr := utils.FormatBalance(action.Value, tconsts.Decimals)
			summaryStr = fmt.Sprintf("%s %s -> %s", amountStr, tconsts.Symbol, codec.MustAddressBech32(tconsts.HRP, action.To))
			if len(action.Memo) > 0 {
				summaryStr += fmt.Sprintf(" (memo: %s)", action.Memo)
			}
		default:
		}
		utils.Outf(
			"%s {{yellow}}%s{{/}} {{yellow}}actor:{{/}} %s {{yellow}}summary (%s):{{/}} [%s] {{yellow}}fee (max %.2f%%):{{/}} %s %s {{yellow}}consumed:{{/}} [%s]\n",
			"✅",
			tx.ID(),
			codec.MustAddressBech32(tconsts.HRP, actor),
			reflect.TypeOf(act),
			summaryStr,
			float64(result.Fee)/float64(tx.Base.MaxFee)*100,
			utils.FormatBalance(result.Fee, tconsts.Decimals),
			tconsts.Symbol,
			cli.ParseDimensions(result.Units),
		)
	}
}
