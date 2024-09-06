// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

//nolint:lll
package cmd

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	hutils "github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
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
		txId, err := sendAndWait(ctx, []chain.Action{&actions.SequencerMsg{
			Data:        []byte{0x00, 0x01, 0x02},
			ChainId:     []byte("nkit"),
			FromAddress: recipient,
		}}, cli, scli, tcli, factory, priorityFee, true)

		hutils.Outf("{{green}}txId:{{/}} %s\n", txId)

		return err
	},
}