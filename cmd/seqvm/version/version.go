// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/version"
)

func init() {
	cobra.EnablePrefixMatching = true
}

// NewCommand implements "seqvm version" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints out the verson",
		RunE:  versionFunc,
	}
	return cmd
}

func versionFunc(*cobra.Command, []string) error {
	fmt.Printf("%s@%s (%s)\n", consts.Name, version.Version, consts.ID)
	return nil
}
