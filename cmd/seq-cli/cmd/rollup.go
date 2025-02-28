package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var rollupCmd = &cobra.Command{
	Use: "rollupCmd",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var getCertCmd = &cobra.Command{
	Use: "get-cert [chainID] [height]",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return ErrInvalidArgs
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		_, _, _, _, _, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		chainID := args[0]
		blockNumber, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}

		cert, err := tcli.GetCertByChainInfo(ctx, chainID, uint64(blockNumber))
		if err != nil {
			return err
		}
		fmt.Printf("cert info: %+v\n", cert)
		return nil
	},
}

var getToBNonceCmd = &cobra.Command{
	Use: "get-nonce",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		_, _, _, _, _, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		cert, err := tcli.GetHighestSettledToBNonce(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("cert info: %+v\n", cert)
		return nil
	},
}
