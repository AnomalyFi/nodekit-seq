// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"crypto/rand"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/spf13/cobra"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/cli"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/pubsub"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/consts"

	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
)

var spamCmd = &cobra.Command{
	Use: "spam",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var runSpamCmd = &cobra.Command{
	Use: "run",
	RunE: func(*cobra.Command, []string) error {
		var sclient *rpc.WebSocketClient
		var tclient *trpc.JSONRPCClient
		var maxFeeParsed *uint64
		if maxFee >= 0 {
			v := uint64(maxFee)
			maxFeeParsed = &v
		}
		return handler.Root().Spam(maxTxBacklog, maxFeeParsed, randomRecipient,
			func(uri string, networkID uint32, chainID ids.ID) error { // createClient
				tclient = trpc.NewJSONRPCClient(uri, networkID, chainID)
				sc, err := rpc.NewWebSocketClient(uri, rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
				if err != nil {
					return err
				}
				sclient = sc
				return nil
			},
			func(priv *cli.PrivateKey) (chain.AuthFactory, error) { // getFactory
				return auth.NewED25519Factory(ed25519.PrivateKey(priv.Bytes)), nil
			},
			func() (*cli.PrivateKey, error) { // createAccount
				p, err := ed25519.GeneratePrivateKey()
				if err != nil {
					return nil, err
				}
				return &cli.PrivateKey{
					Address: auth.NewED25519Address(p.PublicKey()),
					Bytes:   p[:],
				}, nil
			},
			func(choice int, address string) (uint64, error) { // lookupBalance
				balance, err := tclient.Balance(context.TODO(), address, ids.Empty)
				if err != nil {
					return 0, err
				}
				utils.Outf(
					"%d) {{cyan}}address:{{/}} %s {{cyan}}balance:{{/}} %s %s\n",
					choice,
					address,
					utils.FormatBalance(balance, consts.Decimals),
					consts.Symbol,
				)
				return balance, err
			},
			func(ctx context.Context, chainID ids.ID) (chain.Parser, error) { // getParser
				return tclient.Parser(ctx)
			},
			func(addr codec.Address, amount uint64) []chain.Action { // getTransfer
				return []chain.Action{&actions.Transfer{
					To:    addr,
					Asset: ids.Empty,
					Value: amount,
				}}
			},
			func(cli *rpc.JSONRPCClient, priv *cli.PrivateKey) func(context.Context, uint64) error { // submitDummy
				return func(ictx context.Context, count uint64) error {
					_, err := sendAndWait(ictx, []chain.Action{&actions.Transfer{
						To:    priv.Address,
						Value: count, // prevent duplicate txs
					}}, cli, sclient, tclient, auth.NewED25519Factory(ed25519.PrivateKey(priv.Bytes)), false)
					return err
				}
			},
		)
	},
}

var runLocalFeeMarketCmd = &cobra.Command{
	Use: "localFeeMarket",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}
		kb, err := handler.Root().PromptInt("kb", 2000)
		if err != nil {
			return err
		}
		data := make([]byte, kb*1024)
		if _, err := rand.Read(data); err != nil {
			return err
		}
		chainID := []byte("nkit")
		for i := 0; i < 10000; i++ {
			_, err = sendAndWait(ctx, []chain.Action{&actions.SequencerMsg{
				Data:        data,
				ChainId:     chainID,
				FromAddress: recipient,
				RelayerID:   1,
			}}, cli, scli, tcli, factory, false)
			if err != nil {
				return err
			}
			price, err := cli.NameSpacePrice(ctx, string(chainID))
			if err != nil {
				return err
			}
			utils.Outf("{{yellow}}post tx price:{{/}} %d\n", price)
		}
		return nil
	},
}
