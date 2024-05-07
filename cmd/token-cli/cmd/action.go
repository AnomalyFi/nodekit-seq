// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//nolint:lll
package cmd

import (
	"context"
	"errors"
	"time"

	// "math/rand"
	// "encoding/hex"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/pubsub"
	"github.com/AnomalyFi/hypersdk/rpc"
	hutils "github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	frpc "github.com/AnomalyFi/nodekit-seq/cmd/token-faucet/rpc"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"

	"github.com/AnomalyFi/nodekit-seq/utils"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/spf13/cobra"
)

const (
	//nolint:unused
	dummyBlockAgeThreshold = 25
	//nolint:unused
	dummyHeightThreshold = 3
)

var actionCmd = &cobra.Command{
	Use: "action",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var fundFaucetCmd = &cobra.Command{
	Use: "fund-faucet",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()

		// Get faucet
		faucetURI, err := handler.Root().PromptString("faucet URI", 0, consts.MaxInt)
		if err != nil {
			return err
		}
		fcli := frpc.NewJSONRPCClient(faucetURI)
		faucetAddress, err := fcli.FaucetAddress(ctx)
		if err != nil {
			return err
		}

		// Get clients
		_, priv, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Get balance
		_, decimals, balance, _, err := handler.GetAssetInfo(ctx, tcli, priv.PublicKey(), ids.Empty, true)
		if balance == 0 || err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", decimals, balance, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		pk, err := utils.ParseAddress(faucetAddress)
		if err != nil {
			return err
		}
		if _, _, err = sendAndWait(ctx, nil, &actions.Transfer{
			To:    pk,
			Asset: ids.Empty,
			Value: amount,
		}, cli, scli, tcli, factory, true); err != nil {
			return err
		}
		hutils.Outf("{{green}}funded faucet:{{/}} %s\n", faucetAddress)
		return nil
	},
}

var transferCmd = &cobra.Command{
	Use: "transfer",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Select token to send
		assetID, err := handler.Root().PromptAsset("assetID", true)
		if err != nil {
			return err
		}
		_, decimals, balance, _, err := handler.GetAssetInfo(ctx, tcli, priv.PublicKey(), assetID, true)
		if balance == 0 || err != nil {
			return err
		}

		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", decimals, balance, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.Transfer{
			To:    recipient,
			Asset: assetID,
			Value: amount,
		}, cli, scli, tcli, factory, true)
		return err
	},
}

var createAssetCmd = &cobra.Command{
	Use: "create-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Add symbol to token
		symbol, err := handler.Root().PromptString("symbol", 1, actions.MaxSymbolSize)
		if err != nil {
			return err
		}

		// Add decimal to token
		decimals, err := handler.Root().PromptInt("decimals", actions.MaxDecimals)
		if err != nil {
			return err
		}

		// Add metadata to token
		metadata, err := handler.Root().PromptString("metadata", 1, actions.MaxMetadataSize)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.CreateAsset{
			Symbol:   []byte(symbol),
			Decimals: uint8(decimals), // already constrain above to prevent overflow
			Metadata: []byte(metadata),
		}, cli, scli, tcli, factory, true)
		return err
	},
}

var sequencerMsgCmd = &cobra.Command{
	Use: "sequencer-msg",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// // Add metadata to token
		// promptText := promptui.Prompt{
		// 	Label: "metadata (can be changed later)",
		// 	Validate: func(input string) error {
		// 		if len(input) > actions.MaxMetadataSize {
		// 			return errors.New("input too large")
		// 		}
		// 		return nil
		// 	},
		// }
		// metadata, err := promptText.Run()
		// if err != nil {
		// 	return err
		// }

		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}
		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.SequencerMsg{
			Data:        []byte{0x00, 0x01, 0x02},
			ChainId:     []byte("nkit"),
			FromAddress: recipient,
		}, cli, scli, tcli, factory, true)

		return err
	},
}

var mintAssetCmd = &cobra.Command{
	Use: "mint-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Select token to mint
		assetID, err := handler.Root().PromptAsset("assetID", false)
		if err != nil {
			return err
		}
		exists, symbol, decimals, metadata, supply, owner, warp, err := tcli.Asset(ctx, assetID, false)
		if err != nil {
			return err
		}
		if !exists {
			hutils.Outf("{{red}}%s does not exist{{/}}\n", assetID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		if warp {
			hutils.Outf("{{red}}cannot mint a warped asset{{/}}\n", assetID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		if owner != utils.Address(priv.PublicKey()) {
			hutils.Outf("{{red}}%s is the owner of %s, you are not{{/}}\n", owner, assetID)
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
		hutils.Outf(
			"{{yellow}}symbol:{{/}} %s {{yellow}}decimals:{{/}} %s {{yellow}}metadata:{{/}} %s {{yellow}}supply:{{/}} %d\n",
			string(symbol),
			decimals,
			string(metadata),
			supply,
		)

		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", decimals, consts.MaxUint64-supply, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, _, err = sendAndWait(ctx, nil, &actions.MintAsset{
			Asset: assetID,
			To:    recipient,
			Value: amount,
		}, cli, scli, tcli, factory, true)
		return err
	},
}

func performImport(
	ctx context.Context,
	scli *rpc.JSONRPCClient,
	dcli *rpc.JSONRPCClient,
	dscli *rpc.WebSocketClient,
	dtcli *trpc.JSONRPCClient,
	exportTxID ids.ID,
	factory chain.AuthFactory,
) error {
	// Select TxID (if not provided)
	var err error
	if exportTxID == ids.Empty {
		exportTxID, err = handler.Root().PromptID("export txID")
		if err != nil {
			return err
		}
	}

	// Generate warp signature (as long as >= 80% stake)
	var (
		msg                     *warp.Message
		subnetWeight, sigWeight uint64
	)
	for ctx.Err() == nil {
		msg, subnetWeight, sigWeight, err = scli.GenerateAggregateWarpSignature(ctx, exportTxID)
		if sigWeight >= (subnetWeight*4)/5 && err == nil {
			break
		}
		if err == nil {
			hutils.Outf(
				"{{yellow}}waiting for signature weight:{{/}} %d {{yellow}}observed:{{/}} %d\n",
				subnetWeight,
				sigWeight,
			)
		} else {
			hutils.Outf("{{red}}encountered error:{{/}} %v\n", err)
		}
		cont, err := handler.Root().PromptBool("try again")
		if err != nil {
			return err
		}
		if !cont {
			hutils.Outf("{{red}}exiting...{{/}}\n")
			return nil
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	wt, err := actions.UnmarshalWarpTransfer(msg.UnsignedMessage.Payload)
	if err != nil {
		return err
	}
	outputAssetID := wt.Asset
	if !wt.Return {
		outputAssetID = actions.ImportedAssetID(wt.Asset, msg.SourceChainID)
	}
	hutils.Outf(
		"%s {{yellow}}to:{{/}} %s {{yellow}}source assetID:{{/}} %s {{yellow}}source symbol:{{/}} %s {{yellow}}output assetID:{{/}} %s {{yellow}}value:{{/}} %s {{yellow}}reward:{{/}} %s {{yellow}}return:{{/}} %t\n",
		hutils.ToID(
			msg.UnsignedMessage.Payload,
		),
		utils.Address(wt.To),
		wt.Asset,
		wt.Symbol,
		outputAssetID,
		hutils.FormatBalance(wt.Value, wt.Decimals),
		hutils.FormatBalance(wt.Reward, wt.Decimals),
		wt.Return,
	)
	if wt.SwapIn > 0 {
		_, outSymbol, outDecimals, _, _, _, _, err := dtcli.Asset(ctx, wt.AssetOut, false)
		if err != nil {
			return err
		}
		hutils.Outf(
			"{{yellow}}asset in:{{/}} %s {{yellow}}swap in:{{/}} %s {{yellow}}asset out:{{/}} %s {{yellow}}symbol out:{{/}} %s {{yellow}}swap out:{{/}} %s {{yellow}}swap expiry:{{/}} %d\n",
			outputAssetID,
			hutils.FormatBalance(wt.SwapIn, wt.Decimals),
			wt.AssetOut,
			outSymbol,
			hutils.FormatBalance(wt.SwapOut, outDecimals),
			wt.SwapExpiry,
		)
	}
	hutils.Outf(
		"{{yellow}}signature weight:{{/}} %d {{yellow}}total weight:{{/}} %d\n",
		sigWeight,
		subnetWeight,
	)

	// Select fill
	var fill bool
	if wt.SwapIn > 0 {
		fill, err = handler.Root().PromptBool("fill")
		if err != nil {
			return err
		}
	}
	if !fill && wt.SwapExpiry > time.Now().UnixMilli() {
		return ErrMustFill
	}

	// Generate transaction
	_, _, err = sendAndWait(ctx, msg, &actions.ImportAsset{
		Fill: fill,
	}, dcli, dscli, dtcli, factory, true)
	return err
}

// TODO need to update this
// func performImportMsg(
// 	ctx context.Context,
// 	scli *rpc.JSONRPCClient,
// 	dcli *rpc.JSONRPCClient,
// 	dtcli *trpc.JSONRPCClient,
// 	exportTxID ids.ID,
// 	priv ed25519.PrivateKey,
// 	factory chain.AuthFactory,
// ) error {
// 	// Select TxID (if not provided)
// 	var err error
// 	if exportTxID == ids.Empty {
// 		exportTxID, err = promptID("export txID")
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	// Generate warp signature (as long as >= 80% stake)
// 	var (
// 		msg                     *warp.Message
// 		subnetWeight, sigWeight uint64
// 	)
// 	for ctx.Err() == nil {
// 		msg, subnetWeight, sigWeight, err = scli.GenerateAggregateWarpSignature(ctx, exportTxID)
// 		if sigWeight >= (subnetWeight*4)/5 && err == nil {
// 			break
// 		}
// 		if err == nil {
// 			hutils.Outf(
// 				"{{yellow}}waiting for signature weight:{{/}} %d {{yellow}}observed:{{/}} %d\n",
// 				subnetWeight,
// 				sigWeight,
// 			)
// 		} else {
// 			hutils.Outf("{{red}}encountered error:{{/}} %v\n", err)
// 		}
// 		cont, err := promptBool("try again")
// 		if err != nil {
// 			return err
// 		}
// 		if !cont {
// 			hutils.Outf("{{red}}exiting...{{/}}\n")
// 			return nil
// 		}
// 	}
// 	if ctx.Err() != nil {
// 		return ctx.Err()
// 	}
// 	wt, err := chain.UnmarshalWarpBlock(msg.UnsignedMessage.Payload)
// 	if err != nil {
// 		return err
// 	}
// 	hutils.Outf(
// 		"{{yellow}}Timestamp:{{/}} %d {{yellow}}Height:{{/}} %d\n",
// 		wt.Tmstmp,
// 		wt.Hght,
// 	)

// 	// hutils.Outf(
// 	// 	"%s {{yellow}}to:{{/}} %s {{yellow}}source \n",
// 	// 	hutils.ToID(
// 	// 		msg.UnsignedMessage.Payload,
// 	// 	),
// 	// )
// 	hutils.Outf(
// 		"{{yellow}}signature weight:{{/}} %d {{yellow}}total weight:{{/}} %d\n",
// 		sigWeight,
// 		subnetWeight,
// 	)

// 	// Attempt to send dummy transaction if needed
// 	if err := submitDummy(ctx, dcli, dtcli, priv.PublicKey(), factory); err != nil {
// 		fmt.Println("Error in submit of dummy TX: %w", err)
// 		return err
// 	}
// 	fmt.Println("MAKE IT PAST DUMMY")

// 	// Generate transaction
// 	parser, err := dtcli.Parser(ctx)
// 	if err != nil {
// 		fmt.Println("Error in parser of dummy TX: %w", err)
// 		return err
// 	}

// 	fmt.Println("MAKE IT PAST Parser")

// 	//TODO this is what tests our block validation
// 	submit, tx, _, err := dcli.GenerateTransaction(ctx, parser, msg, &actions.ImportBlockMsg{
// 		Fill: true,
// 	}, factory, true)
// 	if err != nil {
// 		fmt.Println("Error in generation of verification TX: %w", err)
// 		return err
// 	}

// 	fmt.Println("MAKE IT PAST generate")

// 	if err := submit(ctx); err != nil {
// 		fmt.Println("Error in submission of verification TX: %w", err)
// 		return err
// 	}

// 	fmt.Println("MAKE IT PAST submit")

// 	success, err := dtcli.WaitForTransaction(ctx, tx.ID())
// 	if err != nil {
// 		return err
// 	}
// 	printStatus(tx.ID(), success)
// 	return nil
// }

// func submitDummy(
// 	ctx context.Context,
// 	cli *rpc.JSONRPCClient,
// 	tcli *trpc.JSONRPCClient,
// 	dest ed25519.PublicKey,
// 	factory chain.AuthFactory,
// ) error {
// 	var (
// 		logEmitted bool
// 		txsSent    uint64
// 	)
// 	for ctx.Err() == nil {
// 		_, h, t, err := cli.Accepted(ctx)
// 		if err != nil {
// 			return err
// 		}
// 		underHeight := h < dummyHeightThreshold
// 		if underHeight || time.Now().Unix()-t > dummyBlockAgeThreshold {
// 			if underHeight && !logEmitted {
// 				hutils.Outf(
// 					"{{yellow}}waiting for snowman++ activation (needed for AWM)...{{/}}\n",
// 				)
// 				logEmitted = true
// 			}
// 			parser, err := tcli.Parser(ctx)
// 			if err != nil {
// 				return err
// 			}
// 			submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.Transfer{
// 				To:    dest,
// 				Value: txsSent + 1, // prevent duplicate txs
// 			}, factory, false)
// 			if err != nil {
// 				return err
// 			}
// 			if err := submit(ctx); err != nil {
// 				return err
// 			}
// 			if _, err := tcli.WaitForTransaction(ctx, tx.ID()); err != nil {
// 				return err
// 			}
// 			txsSent++
// 			time.Sleep(750 * time.Millisecond)
// 			continue
// 		}
// 		if logEmitted {
// 			hutils.Outf("{{yellow}}snowman++ activated{{/}}\n")
// 		}
// 		return nil
// 	}
// 	return ctx.Err()
// }

var importAssetCmd = &cobra.Command{
	Use: "import-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		currentChainID, _, factory, dcli, dscli, dtcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Select source
		_, uris, err := handler.Root().PromptChain("sourceChainID", set.Of(currentChainID))
		if err != nil {
			return err
		}
		scli := rpc.NewJSONRPCClient(uris[0])

		// Perform import
		return performImport(ctx, scli, dcli, dscli, dtcli, ids.Empty, factory)
	},
}

var exportAssetCmd = &cobra.Command{
	Use: "export-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		currentChainID, priv, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Select token to send
		assetID, err := handler.Root().PromptAsset("assetID", true)
		if err != nil {
			return err
		}
		_, decimals, balance, sourceChainID, err := handler.GetAssetInfo(ctx, tcli, priv.PublicKey(), assetID, true)
		if balance == 0 || err != nil {
			return err
		}

		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", decimals, balance, nil)
		if err != nil {
			return err
		}

		// Determine return
		var ret bool
		if sourceChainID != ids.Empty {
			ret = true
		}

		// Select reward
		reward, err := handler.Root().PromptAmount("reward", decimals, balance-amount, nil)
		if err != nil {
			return err
		}

		// Determine destination
		destination := sourceChainID
		if !ret {
			destination, _, err = handler.Root().PromptChain("destination", set.Of(currentChainID))
			if err != nil {
				return err
			}
		}

		// Determine if swap in
		swap, err := handler.Root().PromptBool("swap on import")
		if err != nil {
			return err
		}
		var (
			swapIn     uint64
			assetOut   ids.ID
			swapOut    uint64
			swapExpiry int64
		)
		if swap {
			swapIn, err = handler.Root().PromptAmount("swap in", decimals, amount, nil)
			if err != nil {
				return err
			}
			assetOut, err = handler.Root().PromptAsset("asset out (on destination)", true)
			if err != nil {
				return err
			}
			uris, err := handler.Root().GetChain(destination)
			if err != nil {
				return err
			}
			networkID, _, _, err := cli.Network(ctx)
			if err != nil {
				return err
			}
			dcli := trpc.NewJSONRPCClient(uris[0], networkID, destination)
			_, decimals, _, _, err := handler.GetAssetInfo(ctx, dcli, priv.PublicKey(), assetOut, false)
			if err != nil {
				return err
			}
			swapOut, err = handler.Root().PromptAmount(
				"swap out (on destination, no decimals)",
				decimals,
				consts.MaxUint64,
				nil,
			)
			if err != nil {
				return err
			}
			swapExpiry, err = handler.Root().PromptTime("swap expiry")
			if err != nil {
				return err
			}
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		success, txID, err := sendAndWait(ctx, nil, &actions.ExportAsset{
			To:          recipient,
			Asset:       assetID,
			Value:       amount,
			Return:      ret,
			Reward:      reward,
			SwapIn:      swapIn,
			AssetOut:    assetOut,
			SwapOut:     swapOut,
			SwapExpiry:  swapExpiry,
			Destination: destination,
		}, cli, scli, tcli, factory, true)
		if err != nil {
			return err
		}
		if !success {
			return errors.New("not successful")
		}

		// Perform import
		imp, err := handler.Root().PromptBool("perform import on destination")
		if err != nil {
			return err
		}
		if imp {
			uris, err := handler.Root().GetChain(destination)
			if err != nil {
				return err
			}
			networkID, _, _, err := cli.Network(ctx)
			if err != nil {
				return err
			}
			dscli, err := rpc.NewWebSocketClient(uris[0], rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
			if err != nil {
				return err
			}
			if err := performImport(ctx, cli, rpc.NewJSONRPCClient(uris[0]), dscli, trpc.NewJSONRPCClient(uris[0], networkID, destination), txID, factory); err != nil {
				return err
			}
		}

		// Ask if user would like to switch to destination chain
		sw, err := handler.Root().PromptBool("switch default chain to destination")
		if err != nil {
			return err
		}
		if !sw {
			return nil
		}
		return handler.Root().StoreDefaultChain(destination)
	},
}

// var exportBlockCmd = &cobra.Command{
// 	Use: "export-block",
// 	RunE: func(*cobra.Command, []string) error {
// 		ctx := context.Background()
// 		_, priv, factory, cli, tcli, err := defaultActor()
// 		if err != nil {
// 			return err
// 		}

// 		//, set.Set[ids.ID]{currentChainID: {}}
// 		sourceChainID, _, err := promptChainNoExclude("sourceChainID")

// 		if err != nil {
// 			return err
// 		}

// 		blockRoot, err := promptID("stateRoot")

// 		if err != nil {
// 			return err
// 		}

// 		parentRoot, err := promptID("parentRoot")

// 		if err != nil {
// 			return err
// 		}

// 		height, err := promptUint("Height")

// 		if err != nil {
// 			return err
// 		}

// 		tmstmp, err := promptTime("Timestamp")

// 		if err != nil {
// 			return err
// 		}

// 		// Determine destination
// 		destination := sourceChainID
// 		// if !ret {
// 		// 	destination, _, err = promptChain("destination", set.Set[ids.ID]{currentChainID: {}})
// 		// 	if err != nil {
// 		// 		return err
// 		// 	}
// 		// }

// 		// Determine if swap in

// 		// Confirm action
// 		cont, err := promptContinue()
// 		if !cont || err != nil {
// 			return err
// 		}

// 		// Attempt to send dummy transaction if needed
// 		if err := submitDummy(ctx, cli, tcli, priv.PublicKey(), factory); err != nil {
// 			return err
// 		}

// 		fmt.Println("GOT TO HERE")

// 		// Generate transaction
// 		parser, err := tcli.Parser(ctx)
// 		if err != nil {
// 			return err
// 		}
// 		submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.ExportBlockMsg{
// 			Prnt:        parentRoot,
// 			Tmstmp:      tmstmp,
// 			Hght:        height,
// 			StateRoot:   blockRoot,
// 			Destination: destination,
// 		}, factory, false)
// 		if err != nil {
// 			fmt.Errorf("Error in generation of TX: %w", err)
// 			return err
// 		}
// 		if err := submit(ctx); err != nil {
// 			fmt.Errorf("Error in submit of TX: %w", err)
// 			return err
// 		}
// 		success, err := tcli.WaitForTransaction(ctx, tx.ID())
// 		if err != nil {
// 			return err
// 		}
// 		printStatus(tx.ID(), success)

// 		// Perform import
// 		imp, err := promptBool("perform import on destination for block")
// 		if err != nil {
// 			return err
// 		}
// 		if imp {
// 			uris, err := GetChain(destination)
// 			if err != nil {
// 				return err
// 			}
// 			if err := performImportMsg(ctx, cli, rpc.NewJSONRPCClient(uris[0]), trpc.NewJSONRPCClient(uris[0], destination), tx.ID(), priv, factory); err != nil {
// 				return err
// 			}
// 		}

// 		// Ask if user would like to switch to destination chain
// 		sw, err := promptBool("switch default chain to destination")
// 		if err != nil {
// 			return err
// 		}
// 		if !sw {
// 			return nil
// 		}
// 		return StoreDefault(defaultChainKey, destination[:])
// 	},
// }
