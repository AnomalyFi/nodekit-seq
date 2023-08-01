// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//nolint:lll
package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	// "math/rand"
	// "encoding/hex"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/crypto"
	"github.com/AnomalyFi/hypersdk/rpc"
	hutils "github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
	"github.com/AnomalyFi/nodekit-seq/utils"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const (
	dummyBlockAgeThreshold = 25
	dummyHeightThreshold   = 3
)

var actionCmd = &cobra.Command{
	Use: "action",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var transferCmd = &cobra.Command{
	Use: "transfer",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, tcli, err := defaultActor()
		if err != nil {
			return err
		}

		// Select token to send
		assetID, err := promptAsset("assetID", true)
		if err != nil {
			return err
		}
		balance, _, err := getAssetInfo(ctx, tcli, priv.PublicKey(), assetID, true)
		if balance == 0 || err != nil {
			return err
		}

		// Select recipient
		recipient, err := promptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := promptAmount("amount", assetID, balance, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := promptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		parser, err := tcli.Parser(ctx)
		if err != nil {
			return err
		}
		submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.Transfer{
			To:    recipient,
			Asset: assetID,
			Value: amount,
		}, factory, false)
		if err != nil {
			return err
		}
		if err := submit(ctx); err != nil {
			return err
		}
		success, err := tcli.WaitForTransaction(ctx, tx.ID())
		if err != nil {
			return err
		}
		printStatus(tx.ID(), success)
		return nil
	},
}

var createAssetCmd = &cobra.Command{
	Use: "create-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, tcli, err := defaultActor()
		if err != nil {
			return err
		}

		// Add metadata to token
		promptText := promptui.Prompt{
			Label: "metadata (can be changed later)",
			Validate: func(input string) error {
				if len(input) > actions.MaxMetadataSize {
					return errors.New("input too large")
				}
				return nil
			},
		}
		metadata, err := promptText.Run()
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := promptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		parser, err := tcli.Parser(ctx)
		if err != nil {
			return err
		}
		submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.CreateAsset{
			Metadata: []byte(metadata),
		}, factory, false)
		if err != nil {
			return err
		}
		if err := submit(ctx); err != nil {
			return err
		}
		success, err := tcli.WaitForTransaction(ctx, tx.ID())
		if err != nil {
			return err
		}
		printStatus(tx.ID(), success)
		return nil
	},
}

var sequencerMsgCmd = &cobra.Command{
	Use: "sequencer-msg",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, tcli, err := defaultActor()
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

		recipient, err := promptAddress("recipient")
		if err != nil {
			return err
		}
		// Confirm action
		cont, err := promptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		parser, err := tcli.Parser(ctx)
		if err != nil {
			return err
		}
		submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.SequencerMsg{
			Data:        []byte{0x00, 0x01, 0x02},
			ChainId:     []byte{0x00},
			FromAddress: recipient,
		}, factory, false)
		if err != nil {
			return err
		}
		if err := submit(ctx); err != nil {
			return err
		}
		success, err := tcli.WaitForTransaction(ctx, tx.ID())
		if err != nil {
			return err
		}
		printStatus(tx.ID(), success)
		return nil
	},
}

var mintAssetCmd = &cobra.Command{
	Use: "mint-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, tcli, err := defaultActor()
		if err != nil {
			return err
		}

		// Select token to mint
		assetID, err := promptAsset("assetID", false)
		if err != nil {
			return err
		}
		exists, metadata, supply, owner, warp, err := tcli.Asset(ctx, assetID)
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
			"{{yellow}}metadata:{{/}} %s {{yellow}}supply:{{/}} %d\n",
			string(metadata),
			supply,
		)

		// Select recipient
		recipient, err := promptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := promptAmount("amount", assetID, consts.MaxUint64-supply, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := promptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		parser, err := tcli.Parser(ctx)
		if err != nil {
			return err
		}
		submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.MintAsset{
			Asset: assetID,
			To:    recipient,
			Value: amount,
		}, factory, false)
		if err != nil {
			return err
		}
		if err := submit(ctx); err != nil {
			return err
		}
		success, err := tcli.WaitForTransaction(ctx, tx.ID())
		if err != nil {
			return err
		}
		printStatus(tx.ID(), success)
		return nil
	},
}

func performImport(
	ctx context.Context,
	scli *rpc.JSONRPCClient,
	dcli *rpc.JSONRPCClient,
	dtcli *trpc.JSONRPCClient,
	exportTxID ids.ID,
	priv crypto.PrivateKey,
	factory chain.AuthFactory,
) error {
	// Select TxID (if not provided)
	var err error
	if exportTxID == ids.Empty {
		exportTxID, err = promptID("export txID")
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
		cont, err := promptBool("try again")
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
		"%s {{yellow}}to:{{/}} %s {{yellow}}source assetID:{{/}} %s {{yellow}}output assetID:{{/}} %s {{yellow}}value:{{/}} %s {{yellow}}reward:{{/}} %s {{yellow}}return:{{/}} %t\n",
		hutils.ToID(
			msg.UnsignedMessage.Payload,
		),
		utils.Address(wt.To),
		assetString(wt.Asset),
		assetString(outputAssetID),
		valueString(outputAssetID, wt.Value),
		valueString(outputAssetID, wt.Reward),
		wt.Return,
	)
	if wt.SwapIn > 0 {
		hutils.Outf(
			"{{yellow}}asset in:{{/}} %s {{yellow}}swap in:{{/}} %s {{yellow}}asset out:{{/}} %s {{yellow}}swap out:{{/}} %s {{yellow}}swap expiry:{{/}} %d\n",
			assetString(outputAssetID),
			valueString(
				outputAssetID,
				wt.SwapIn,
			),
			assetString(wt.AssetOut),
			valueString(wt.AssetOut, wt.SwapOut),
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
		fill, err = promptBool("fill")
		if err != nil {
			return err
		}
	}
	if !fill && wt.SwapExpiry > time.Now().Unix() {
		return ErrMustFill
	}

	// Attempt to send dummy transaction if needed
	if err := submitDummy(ctx, dcli, dtcli, priv.PublicKey(), factory); err != nil {
		return err
	}

	// Generate transaction
	parser, err := dtcli.Parser(ctx)
	if err != nil {
		return err
	}
	submit, tx, _, err := dcli.GenerateTransaction(ctx, parser, msg, &actions.ImportAsset{
		Fill: fill,
	}, factory, false)
	if err != nil {
		return err
	}
	if err := submit(ctx); err != nil {
		return err
	}
	success, err := dtcli.WaitForTransaction(ctx, tx.ID())
	if err != nil {
		return err
	}
	printStatus(tx.ID(), success)
	return nil
}

func performImportMsg(
	ctx context.Context,
	scli *rpc.JSONRPCClient,
	dcli *rpc.JSONRPCClient,
	dtcli *trpc.JSONRPCClient,
	exportTxID ids.ID,
	priv crypto.PrivateKey,
	factory chain.AuthFactory,
) error {
	// Select TxID (if not provided)
	var err error
	if exportTxID == ids.Empty {
		exportTxID, err = promptID("export txID")
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
		cont, err := promptBool("try again")
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
	wt, err := chain.UnmarshalWarpBlock(msg.UnsignedMessage.Payload)
	if err != nil {
		return err
	}
	hutils.Outf(
		"{{yellow}}Timestamp:{{/}} %d {{yellow}}Height:{{/}} %d\n",
		wt.Tmstmp,
		wt.Hght,
	)

	// hutils.Outf(
	// 	"%s {{yellow}}to:{{/}} %s {{yellow}}source \n",
	// 	hutils.ToID(
	// 		msg.UnsignedMessage.Payload,
	// 	),
	// )
	hutils.Outf(
		"{{yellow}}signature weight:{{/}} %d {{yellow}}total weight:{{/}} %d\n",
		sigWeight,
		subnetWeight,
	)

	// Attempt to send dummy transaction if needed
	if err := submitDummy(ctx, dcli, dtcli, priv.PublicKey(), factory); err != nil {
		fmt.Println("Error in submit of dummy TX: %w", err)
		return err
	}
	fmt.Println("MAKE IT PAST DUMMY")

	// Generate transaction
	parser, err := dtcli.Parser(ctx)
	if err != nil {
		fmt.Println("Error in parser of dummy TX: %w", err)
		return err
	}

	fmt.Println("MAKE IT PAST Parser")

	//TODO this is what tests our block validation
	submit, tx, _, err := dcli.GenerateTransaction(ctx, parser, msg, &actions.ImportBlockMsg{}, factory, true)
	if err != nil {
		fmt.Println("Error in generation of verification TX: %w", err)
		return err
	}

	fmt.Println("MAKE IT PAST generate")

	if err := submit(ctx); err != nil {
		fmt.Println("Error in submission of verification TX: %w", err)
		return err
	}

	fmt.Println("MAKE IT PAST submit")

	success, err := dtcli.WaitForTransaction(ctx, tx.ID())
	if err != nil {
		return err
	}
	printStatus(tx.ID(), success)
	return nil
}

func submitDummy(
	ctx context.Context,
	cli *rpc.JSONRPCClient,
	tcli *trpc.JSONRPCClient,
	dest crypto.PublicKey,
	factory chain.AuthFactory,
) error {
	var (
		logEmitted bool
		txsSent    uint64
	)
	for ctx.Err() == nil {
		_, h, t, err := cli.Accepted(ctx)
		if err != nil {
			return err
		}
		underHeight := h < dummyHeightThreshold
		if underHeight || time.Now().Unix()-t > dummyBlockAgeThreshold {
			if underHeight && !logEmitted {
				hutils.Outf(
					"{{yellow}}waiting for snowman++ activation (needed for AWM)...{{/}}\n",
				)
				logEmitted = true
			}
			parser, err := tcli.Parser(ctx)
			if err != nil {
				return err
			}
			submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.Transfer{
				To:    dest,
				Value: txsSent + 1, // prevent duplicate txs
			}, factory, false)
			if err != nil {
				return err
			}
			if err := submit(ctx); err != nil {
				return err
			}
			if _, err := tcli.WaitForTransaction(ctx, tx.ID()); err != nil {
				return err
			}
			txsSent++
			time.Sleep(750 * time.Millisecond)
			continue
		}
		if logEmitted {
			hutils.Outf("{{yellow}}snowman++ activated{{/}}\n")
		}
		return nil
	}
	return ctx.Err()
}

var importAssetCmd = &cobra.Command{
	Use: "import-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		currentChainID, priv, factory, dcli, dtcli, err := defaultActor()
		if err != nil {
			return err
		}

		// Select source
		_, uris, err := promptChain("sourceChainID", set.Set[ids.ID]{currentChainID: {}})
		if err != nil {
			return err
		}
		scli := rpc.NewJSONRPCClient(uris[0])

		// Perform import
		return performImport(ctx, scli, dcli, dtcli, ids.Empty, priv, factory)
	},
}

var exportAssetCmd = &cobra.Command{
	Use: "export-asset",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		currentChainID, priv, factory, cli, tcli, err := defaultActor()
		if err != nil {
			return err
		}

		// Select token to send
		assetID, err := promptAsset("assetID", true)
		if err != nil {
			return err
		}
		balance, sourceChainID, err := getAssetInfo(ctx, tcli, priv.PublicKey(), assetID, true)
		if balance == 0 || err != nil {
			return err
		}

		// Select recipient
		recipient, err := promptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := promptAmount("amount", assetID, balance, nil)
		if err != nil {
			return err
		}

		// Determine return
		var ret bool
		if sourceChainID != ids.Empty {
			ret = true
		}

		// Select reward
		reward, err := promptAmount("reward", assetID, balance-amount, nil)
		if err != nil {
			return err
		}

		// Determine destination
		destination := sourceChainID
		if !ret {
			destination, _, err = promptChain("destination", set.Set[ids.ID]{currentChainID: {}})
			if err != nil {
				return err
			}
		}

		// Determine if swap in
		swap, err := promptBool("swap on import")
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
			swapIn, err = promptAmount("swap in", assetID, amount, nil)
			if err != nil {
				return err
			}
			assetOut, err = promptAsset("asset out (on destination)", true)
			if err != nil {
				return err
			}
			swapOut, err = promptAmount(
				"swap out (on destination)",
				assetOut,
				consts.MaxUint64,
				nil,
			)
			if err != nil {
				return err
			}
			swapExpiry, err = promptTime("swap expiry")
			if err != nil {
				return err
			}
		}

		// Confirm action
		cont, err := promptContinue()
		if !cont || err != nil {
			return err
		}

		// Attempt to send dummy transaction if needed
		if err := submitDummy(ctx, cli, tcli, priv.PublicKey(), factory); err != nil {
			return err
		}

		// Generate transaction
		parser, err := tcli.Parser(ctx)
		if err != nil {
			return err
		}
		submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.ExportAsset{
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
		}, factory, false)
		if err != nil {
			return err
		}
		if err := submit(ctx); err != nil {
			return err
		}
		success, err := tcli.WaitForTransaction(ctx, tx.ID())
		if err != nil {
			return err
		}
		printStatus(tx.ID(), success)

		// Perform import
		imp, err := promptBool("perform import on destination")
		if err != nil {
			return err
		}
		if imp {
			uris, err := GetChain(destination)
			if err != nil {
				return err
			}
			if err := performImport(ctx, cli, rpc.NewJSONRPCClient(uris[0]), trpc.NewJSONRPCClient(uris[0], destination), tx.ID(), priv, factory); err != nil {
				return err
			}
		}

		// Ask if user would like to switch to destination chain
		sw, err := promptBool("switch default chain to destination")
		if err != nil {
			return err
		}
		if !sw {
			return nil
		}
		return StoreDefault(defaultChainKey, destination[:])
	},
}

var exportBlockCmd = &cobra.Command{
	Use: "export-block",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, priv, factory, cli, tcli, err := defaultActor()
		if err != nil {
			return err
		}

		//, set.Set[ids.ID]{currentChainID: {}}
		sourceChainID, _, err := promptChainNoExclude("sourceChainID")

		if err != nil {
			return err
		}

		blockRoot, err := promptID("stateRoot")

		if err != nil {
			return err
		}

		parentRoot, err := promptID("parentRoot")

		if err != nil {
			return err
		}

		height, err := promptUint("Height")

		if err != nil {
			return err
		}

		tmstmp, err := promptTime("Timestamp")

		if err != nil {
			return err
		}

		// Determine destination
		destination := sourceChainID
		// if !ret {
		// 	destination, _, err = promptChain("destination", set.Set[ids.ID]{currentChainID: {}})
		// 	if err != nil {
		// 		return err
		// 	}
		// }

		// Determine if swap in

		// Confirm action
		cont, err := promptContinue()
		if !cont || err != nil {
			return err
		}

		// Attempt to send dummy transaction if needed
		if err := submitDummy(ctx, cli, tcli, priv.PublicKey(), factory); err != nil {
			return err
		}

		fmt.Println("GOT TO HERE")

		// Generate transaction
		parser, err := tcli.Parser(ctx)
		if err != nil {
			return err
		}
		submit, tx, _, err := cli.GenerateTransaction(ctx, parser, nil, &actions.ExportBlockMsg{
			Prnt:        parentRoot,
			Tmstmp:      tmstmp,
			Hght:        height,
			StateRoot:   blockRoot,
			Destination: destination,
		}, factory, false)
		if err != nil {
			fmt.Errorf("Error in generation of TX: %w", err)
			return err
		}
		if err := submit(ctx); err != nil {
			fmt.Errorf("Error in submit of TX: %w", err)
			return err
		}
		success, err := tcli.WaitForTransaction(ctx, tx.ID())
		if err != nil {
			return err
		}
		printStatus(tx.ID(), success)

		// Perform import
		imp, err := promptBool("perform import on destination for block")
		if err != nil {
			return err
		}
		if imp {
			uris, err := GetChain(destination)
			if err != nil {
				return err
			}
			if err := performImportMsg(ctx, cli, rpc.NewJSONRPCClient(uris[0]), trpc.NewJSONRPCClient(uris[0], destination), tx.ID(), priv, factory); err != nil {
				return err
			}
		}

		// Ask if user would like to switch to destination chain
		sw, err := promptBool("switch default chain to destination")
		if err != nil {
			return err
		}
		if !sw {
			return nil
		}
		return StoreDefault(defaultChainKey, destination[:])
	},
}
