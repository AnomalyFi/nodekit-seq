// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "token-cli" implements tokenvm client operation interface.
package main

import (
	"os"

	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/anomalyFi/nodekit-seq/cmd/token-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		utils.Outf("{{red}}token-cli exited with error:{{/}} %+v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
