// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package registry

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/ava-labs/avalanchego/utils/wrappers"

	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/consts"
)

// Setup types
func init() {
	consts.ActionRegistry = codec.NewTypeParser[chain.Action]()
	consts.AuthRegistry = codec.NewTypeParser[chain.Auth]()

	errs := &wrappers.Errs{}
	errs.Add(
		// When registering new actions, ALWAYS make sure to append at the end.
		consts.ActionRegistry.Register((&actions.Transfer{}).GetTypeID(), actions.UnmarshalTransfer, false),
		consts.ActionRegistry.Register((&actions.SequencerMsg{}).GetTypeID(), actions.UnmarshalSequencerMsg, false),
		consts.ActionRegistry.Register((&actions.RollupRegistration{}).GetTypeID(), actions.UnmarshalRollupRegister, false),
		consts.ActionRegistry.Register((&actions.EpochExit{}).GetTypeID(), actions.UnmarshalEpochExit, false),
		consts.ActionRegistry.Register((&actions.Auction{}).GetTypeID(), actions.UnmarshalAuction, false),
		// When registering new auth, ALWAYS make sure to append at the end.
		consts.AuthRegistry.Register((&auth.ED25519{}).GetTypeID(), auth.UnmarshalED25519, false),
		consts.AuthRegistry.Register((&auth.BLS{}).GetTypeID(), auth.UnmarshalBLS, false),
	)
	if errs.Errored() {
		panic(errs.Err)
	}
}
