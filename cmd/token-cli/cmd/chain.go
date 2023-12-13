// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//nolint:lll
package cmd

import (
	"context"
	"fmt"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/spf13/cobra"

	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
)

var chainCmd = &cobra.Command{
	Use: "chain",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var importChainCmd = &cobra.Command{
	Use: "import",
	RunE: func(_ *cobra.Command, args []string) error {
		return handler.Root().ImportChain()
	},
}

var importANRChainCmd = &cobra.Command{
	Use: "import-anr",
	RunE: func(_ *cobra.Command, args []string) error {
		return handler.Root().ImportANR()
	},
}

var importAvalancheOpsChainCmd = &cobra.Command{
	Use: "import-ops [chainID] [path]",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return ErrInvalidArgs
		}

		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		return handler.Root().ImportOps(args[0])
	},
}

var setChainCmd = &cobra.Command{
	Use: "set",
	RunE: func(*cobra.Command, []string) error {
		return handler.Root().SetDefaultChain()
	},
}

var chainInfoCmd = &cobra.Command{
	Use: "info",
	RunE: func(_ *cobra.Command, args []string) error {
		return handler.Root().PrintChainInfo()
	},
}

var watchChainCmd = &cobra.Command{
	Use: "watch",
	RunE: func(_ *cobra.Command, args []string) error {
		var cli *trpc.JSONRPCClient
		return handler.Root().WatchChain(hideTxs, func(uri string, networkID uint32, chainID ids.ID) (chain.Parser, error) {
			fmt.Println("Here is network Id: %d", networkID)
			fmt.Println("Here is uri: %s", uri)

			cli = trpc.NewJSONRPCClient(uri, networkID, chainID)
			return cli.Parser(context.TODO())
		}, func(tx *chain.Transaction, result *chain.Result) {
			if cli == nil {
				// Should never happen
				return
			}
			handleTx(cli, tx, result)
		})
	},
}

var testHeaderCmd = &cobra.Command{
	Use: "test-header",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, _, _, _, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// start, err := handler.Root().PromptTime("start")
		// if err != nil {
		// 	return err
		// }

		// //1698200132261
		// end, err := handler.Root().PromptTime("end")
		// if err != nil {
		// 	return err
		// }

		// // start_time := time.Unix(start, 0)
		// // end_time := time.Unix(end, 0)

		// start := time.Now().Unix()

		// end := time.Now().Unix() - 120

		start := int64(1702426377)
		end := int64(1702426379)

		start_time := start * 1000

		end_time := end * 1000

		res, err := tcli.GetBlockHeadersByStart(ctx, start_time, end_time)

		if err != nil {
			return err
		}

		fmt.Println(res.From)
		fmt.Println(res.Next)
		fmt.Println(res.Prev)

		return nil
	},
}
