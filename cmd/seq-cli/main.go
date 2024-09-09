// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "seq-cli" implements seqvm client operation interface.
package main

import (
	"os"

	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/cmd/seq-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		utils.Outf("{{red}}seq-cli exited with error:{{/}} %+v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
