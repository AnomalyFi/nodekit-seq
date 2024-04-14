// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package registry

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"

	"github.com/anomalyFi/nodekit-seq/actions"
	"github.com/anomalyFi/nodekit-seq/auth"
	"github.com/anomalyFi/nodekit-seq/consts"
)

// Setup types
func init() {
	consts.ActionRegistry = codec.NewTypeParser[chain.Action, *warp.Message]()
	consts.AuthRegistry = codec.NewTypeParser[chain.Auth, *warp.Message]()

	errs := &wrappers.Errs{}
	errs.Add(
		// When registering new actions, ALWAYS make sure to append at the end.
		consts.ActionRegistry.Register((&actions.Transfer{}).GetTypeID(), actions.UnmarshalTransfer, false),

		consts.ActionRegistry.Register((&actions.CreateAsset{}).GetTypeID(), actions.UnmarshalCreateAsset, false),
		consts.ActionRegistry.Register((&actions.MintAsset{}).GetTypeID(), actions.UnmarshalMintAsset, false),
		consts.ActionRegistry.Register((&actions.BurnAsset{}).GetTypeID(), actions.UnmarshalBurnAsset, false),

		consts.ActionRegistry.Register((&actions.CreateOrder{}).GetTypeID(), actions.UnmarshalCreateOrder, false),
		consts.ActionRegistry.Register((&actions.FillOrder{}).GetTypeID(), actions.UnmarshalFillOrder, false),
		consts.ActionRegistry.Register((&actions.CloseOrder{}).GetTypeID(), actions.UnmarshalCloseOrder, false),

		consts.ActionRegistry.Register((&actions.ImportAsset{}).GetTypeID(), actions.UnmarshalImportAsset, true),
		consts.ActionRegistry.Register((&actions.ExportAsset{}).GetTypeID(), actions.UnmarshalExportAsset, false),

		consts.ActionRegistry.Register((&actions.Transact{}).GetTypeID(), actions.UnmarshalTransact, false),
		consts.ActionRegistry.Register((&actions.Deploy{}).GetTypeID(), actions.UnmarshalDeploy, false),

		// When registering new auth, ALWAYS make sure to append at the end.
		consts.AuthRegistry.Register((&auth.ED25519{}).GetTypeID(), auth.UnmarshalED25519, false),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
