// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"github.com/AnomalyFi/nodekit-seq/consts"
	ametrics "github.com/ava-labs/avalanchego/api/metrics"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	createAsset prometheus.Counter
	mintAsset   prometheus.Counter
	burnAsset   prometheus.Counter
	//modifyAsset prometheus.Counter

	transfer prometheus.Counter

	importAsset  prometheus.Counter
	exportAsset  prometheus.Counter
	importBlock  prometheus.Counter
	exportBlock  prometheus.Counter
	sequencerMsg prometheus.Counter
}

func newMetrics(gatherer ametrics.MultiGatherer) (*metrics, error) {
	m := &metrics{
		createAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "create_asset",
			Help:      "number of create asset actions",
		}),
		mintAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "mint_asset",
			Help:      "number of mint asset actions",
		}),
		burnAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "burn_asset",
			Help:      "number of burn asset actions",
		}),
		// modifyAsset: prometheus.NewCounter(prometheus.CounterOpts{
		// 	Namespace: "actions",
		// 	Name:      "modify_asset",
		// 	Help:      "number of modify asset actions",
		// }),
		transfer: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "transfer",
			Help:      "number of transfer actions",
		}),
		importAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "import_asset",
			Help:      "number of import asset actions",
		}),
		exportAsset: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "export_asset",
			Help:      "number of export asset actions",
		}),
		importBlock: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "import_block",
			Help:      "number of import block actions",
		}),
		exportBlock: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "export_block",
			Help:      "number of export block actions",
		}),
		sequencerMsg: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "sequencer_msg",
			Help:      "number of sequencer msg actions",
		}),
	}
	r := prometheus.NewRegistry()
	errs := wrappers.Errs{}
	errs.Add(
		r.Register(m.createAsset),
		r.Register(m.mintAsset),
		r.Register(m.burnAsset),
		//r.Register(m.modifyAsset),

		r.Register(m.transfer),

		r.Register(m.importBlock),
		r.Register(m.exportBlock),
		r.Register(m.importAsset),
		r.Register(m.exportAsset),
		r.Register(m.sequencerMsg),
		gatherer.Register(consts.Name, r),
	)
	return m, errs.Err
}
