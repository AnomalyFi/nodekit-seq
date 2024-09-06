// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/spf13/cobra"

	"github.com/AnomalyFi/hypersdk/cli"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/auth"

	tconsts "github.com/AnomalyFi/nodekit-seq/consts"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
)

var keyCmd = &cobra.Command{
	Use: "key",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var genKeyCmd = &cobra.Command{
	Use: "generate",
	RunE: func(*cobra.Command, []string) error {
		p, err := ed25519.GeneratePrivateKey()
		if err != nil {
			return err
		}
		priv := &cli.PrivateKey{
			Address: auth.NewED25519Address(p.PublicKey()),
			Bytes:   p[:],
		}
		if err := handler.h.StoreKey(priv); err != nil {
			return err
		}
		if err := handler.h.StoreDefaultKey(priv.Address); err != nil {
			return err
		}
		utils.Outf(
			"{{green}}created address:{{/}} %s",
			codec.MustAddressBech32(tconsts.HRP, priv.Address),
		)
		return nil
	},
}

var importKeyCmd = &cobra.Command{
	Use: "import [path]",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return ErrInvalidArgs
		}
		return nil
	},
	RunE: func(_ *cobra.Command, args []string) error {
		p, err := utils.LoadBytes(args[0], ed25519.PrivateKeyLen)
		if err != nil {
			return err
		}
		pk := ed25519.PrivateKey(p)
		priv := &cli.PrivateKey{
			Address: auth.NewED25519Address(pk.PublicKey()),
			Bytes:   p,
		}
		if err := handler.h.StoreKey(priv); err != nil {
			return err
		}
		if err := handler.h.StoreDefaultKey(priv.Address); err != nil {
			return err
		}
		utils.Outf(
			"{{green}}imported address:{{/}} %s",
			codec.MustAddressBech32(tconsts.HRP, priv.Address),
		)
		return nil
	},
}

func lookupSetKeyBalance(choice int, address string, uri string, networkID uint32, chainID ids.ID) error {
	// TODO: just load once
	cli := trpc.NewJSONRPCClient(uri, networkID, chainID)
	balance, err := cli.Balance(context.TODO(), address)
	if err != nil {
		return err
	}
	utils.Outf(
		"%d) {{cyan}}address:{{/}} %s {{cyan}}balance:{{/}} %s %s\n",
		choice,
		address,
		utils.FormatBalance(balance, tconsts.Decimals),
		tconsts.Symbol,
	)
	return nil
}

var setKeyCmd = &cobra.Command{
	Use: "set",
	RunE: func(*cobra.Command, []string) error {
		return handler.Root().SetKey(lookupSetKeyBalance)
	},
}

func lookupKeyBalance(addr codec.Address, uri string, networkID uint32, chainID ids.ID, _ ids.ID) error {
	_, err := handler.GetBalance(
		context.TODO(), trpc.NewJSONRPCClient(uri, networkID, chainID),
		addr)
	return err
}

var balanceKeyCmd = &cobra.Command{
	Use: "balance",
	RunE: func(*cobra.Command, []string) error {
		return handler.Root().Balance(checkAllChains, true, lookupKeyBalance)
	},
}
