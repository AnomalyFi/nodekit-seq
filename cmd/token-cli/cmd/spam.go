// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/rand"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/pubsub"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/consts"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/spf13/cobra"
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
			func(uri string, networkID uint32, chainID ids.ID) {
				tclient = trpc.NewJSONRPCClient(uri, networkID, chainID)
				sc, err := rpc.NewWebSocketClient(uri, rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
				if err != nil {
					panic(err)
				}
				sclient = sc
			},
			func(pk ed25519.PrivateKey) chain.AuthFactory {
				return auth.NewED25519Factory(pk)
			},
			func(choice int, address string) (uint64, error) {
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
			func(ctx context.Context, chainID ids.ID) (chain.Parser, error) {
				return tclient.Parser(ctx)
			},
			func(pk ed25519.PublicKey, amount uint64) chain.Action {
				return &actions.Transfer{
					To:    pk,
					Asset: ids.Empty,
					Value: amount,
				}
			},
			func(cli *rpc.JSONRPCClient, pk ed25519.PrivateKey) func(context.Context, uint64) error {
				return func(ictx context.Context, count uint64) error {
					_, _, err := sendAndWait(ictx, nil, &actions.Transfer{
						To:    pk.PublicKey(),
						Value: count, // prevent duplicate txs
					}, cli, sclient, tclient, auth.NewED25519Factory(pk), false)
					return err
				}
			},
		)
	},
}

var runSpamSequencerMsgCmd = &cobra.Command{
	Use: "smsg",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()

		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		numAddress, err := handler.Root().PromptInt("how many addresses to generate?", 500)
		if err != nil {
			return err
		}

		numMsgs, err := handler.Root().PromptInt("how many msgs to send?", 10000)
		if err != nil {
			return err
		}

		for i := 0; i < numAddress; i++ {
			handler.Root().GenerateKey()
		}

		keys, err := handler.Root().GetKeys()
		if err != nil {
			return err
		}

		for _, k := range keys {
			for i := 0; i < numMsgs; i++ {
				data, err := randomBytes()
				fmt.Printf("data size(byte): %d\n", len(data))
				if err != nil {
					fmt.Println("error genrateing bytes, skipping")
					continue
				}
				_, _, err = sendAndWait(ctx, nil, &actions.SequencerMsg{
					Data:        data,
					ChainId:     []byte("nkit"),
					FromAddress: k.PublicKey(),
				}, cli, scli, tcli, factory, true)
				if err != nil {
					fmt.Println("error submitting tx, skipping")
				}
			}
		}

		return err
	},
}

func randomBytes() ([]byte, error) {
	// 256 kb
	numBytes := rand.Intn(256 * 104)

	b := make([]byte, numBytes)

	_, err := crand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
