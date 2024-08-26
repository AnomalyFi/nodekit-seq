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
	Use: "import-ops [path]",
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
		ctx := context.Background()
		pastBlocks, err := handler.Root().PromptBool("streaming from past blocks?")
		if err != nil {
			return err
		}
		var startBlock uint64
		if pastBlocks {
			startBlock, err = handler.Root().PromptUint64("start block")
			if err != nil {
				return err
			}
			_, _, _, cli, _, bcli, err := handler.DefaultActor()
			if err != nil {
				return err
			}
			_, lastAccepted, _, err := cli.Accepted(ctx)
			if err != nil {
				return err
			}
			acceptedWindow, err := bcli.GetAcceptedBlockWindow(ctx)
			if err != nil {
				return err
			}
			if lastAccepted > uint64(acceptedWindow) && lastAccepted-uint64(acceptedWindow) > startBlock {
				return fmt.Errorf("start block is too old")
			}
		}

		var cli *trpc.JSONRPCClient
		return handler.Root().WatchChain(hideTxs, pastBlocks, startBlock, func(uri string, networkID uint32, chainID ids.ID) (chain.Parser, error) {
			cli = trpc.NewJSONRPCClient(uri, networkID, chainID)
			return cli.Parser(context.TODO())
		}, func(tx *chain.Transaction, result *chain.Result) {
			if cli == nil {
				// Should never happen
				return
			}
			handleTx(tx, result)
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

		start := int64(1702502928)
		end := int64(1702502930)

		startTime := start * 1000

		endTime := end * 1000

		res, err := tcli.GetBlockHeadersByStartTimeStamp(ctx, startTime, endTime)
		if err != nil {
			return err
		}

		fmt.Println(res.From)
		fmt.Println(res.Next)
		fmt.Println(res.Prev)

		return nil
	},
}

var anchorsCmd = &cobra.Command{
	Use: "anchors",
	RunE: func(_ *cobra.Command, args []string) error {
		ctx := context.Background()
		_, _, _, _, _, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		namespaces, infos, err := tcli.RegisteredAnchors(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("num of anchors registered: %d\n", len(namespaces))
		for i := 0; i < len(namespaces); i++ {
			fmt.Printf("%s: %+v\n", string(namespaces[i]), infos[i])
		}

		return nil
	},
}

var replaceAnchorCmd = &cobra.Command{
	Use: "replace-anchor",
	RunE: func(_ *cobra.Command, args []string) error {
		anchorURL, err := handler.Root().PromptString("anchor addr", 0, 500)
		if err != nil {
			return err
		}

		ctx := context.Background()
		_, _, _, hcli, _, _, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		replaced, err := hcli.ReplaceAnchor(ctx, anchorURL)
		if err != nil {
			return err
		}
		if replaced {
			fmt.Printf("replaced anchor to %s", anchorURL)
		} else {
			fmt.Println("unable to replace anchor")
		}
		return nil
	},
}
