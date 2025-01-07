// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//nolint:lll
package cmd

import (
	"context"
	"encoding/binary"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/chain"
	hconsts "github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/crypto/bls"
	hrpc "github.com/AnomalyFi/hypersdk/rpc"
	hutils "github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/consts"

	"github.com/spf13/cobra"
)

var priorityFee uint64 = 0

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
		_, priv, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Get balance info
		balance, err := handler.GetBalance(ctx, tcli, priv.Address)
		if balance == 0 || err != nil {
			return err
		}

		// Select recipient
		recipient, err := handler.Root().PromptAddress("recipient")
		if err != nil {
			return err
		}

		// Select amount
		amount, err := handler.Root().PromptAmount("amount", consts.Decimals, balance, nil)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		// Generate transaction
		_, err = sendAndWait(ctx, []chain.Action{&actions.Transfer{
			To:    recipient,
			Value: amount,
		}}, cli, scli, tcli, factory, priorityFee, true)
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
		txID, err := sendAndWait(ctx, []chain.Action{&actions.SequencerMsg{
			Data:        []byte{0x00, 0x01, 0x02},
			ChainID:     []byte("nkit"),
			FromAddress: recipient,
		}}, cli, scli, tcli, factory, priorityFee, true)

		hutils.Outf("{{green}}txId:{{/}} %s\n", txID)

		return err
	},
}

var rollupCmd = &cobra.Command{
	Use: "rollup-register",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		namespaceStr, err := handler.Root().PromptString("namespace", 0, 8)
		if err != nil {
			return err
		}
		namespace := []byte(namespaceStr)
		feeRecipient, err := handler.Root().PromptAddress("feeRecipient")
		if err != nil {
			return err
		}

		op, err := handler.Root().PromptChoice("(0)create (1)exit (2)update", 3)
		if err != nil {
			return err
		}

		e, _ := cli.GetCurrentEpoch()
		info := hactions.RollupInfo{
			FeeRecipient:        feeRecipient,
			Namespace:           namespace,
			AuthoritySEQAddress: feeRecipient,
			SequencerPublicKey:  feeRecipient[:],
			StartEpoch:          e + 10,
			ExitEpoch:           0,
		}
		// Generate transaction
		_, err = sendAndWait(ctx, []chain.Action{&actions.RollupRegistration{
			Namespace: namespace,
			Info:      info,
			OpCode:    op,
		}}, cli, scli, tcli, factory, 0, true)
		return err
	},
}

var auctionCmd = &cobra.Command{
	Use: "auction",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		bidPrice, err := handler.Root().PromptAmount("bid price", consts.Decimals, 0, nil)
		if err != nil {
			return err
		}
		_, hght, _, err := cli.Accepted(ctx)
		if err != nil {
			return err
		}

		epochNumber := hght/12*hconsts.MillisecondsPerSecond + 1

		p, err := bls.GeneratePrivateKey()
		if err != nil {
			return err
		}
		builderSEQAddress := auth.NewBLSAddress(bls.PublicFromPrivateKey(p))
		pubKeyBytes := bls.PublicKeyToBytes(bls.PublicFromPrivateKey(p))

		builderMsg := make([]byte, 16)
		binary.BigEndian.PutUint64(builderMsg[:8], epochNumber)
		binary.BigEndian.PutUint64(builderMsg[8:], bidPrice)
		builderMsg = append(builderMsg, pubKeyBytes...)

		sig := bls.Sign(builderMsg, p)

		// Generate transaction
		action := []chain.Action{
			&actions.Transfer{
				To:    builderSEQAddress,
				Value: bidPrice * 2,
			},
			&actions.Auction{
				AuctionInfo: actions.AuctionInfo{
					EpochNumber:       epochNumber,
					BidPrice:          bidPrice,
					BuilderSEQAddress: builderSEQAddress,
				},
				BuilderPublicKey: pubKeyBytes,
				BuilderSignature: bls.SignatureToBytes(sig),
			},
		}

		_, err = sendAndWait(ctx, action, cli, scli, tcli, factory, 0, true)
		return err
	},
}

var updateArcadiaURL = &cobra.Command{
	Use: "updateArcadiaURL",
	RunE: func(*cobra.Command, []string) error {
		// ctx := context.Background()
		str, err := handler.Root().PromptString("val rpc url", 1, 100)
		if err != nil {
			return err
		}
		valCLI := hrpc.NewJSONRPCValClient(str)
		str, err = handler.Root().PromptString("new arcadia url", 1, 150)
		if err != nil {
			return err
		}
		return valCLI.UpdateArcadiaURL(str)
	},
}
