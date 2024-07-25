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

	transfer prometheus.Counter

	sequencerMsg prometheus.Counter

	oracle prometheus.Counter

	deploy   prometheus.Counter
	transact prometheus.Counter
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
		transfer: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "transfer",
			Help:      "number of transfer actions",
		}),

		sequencerMsg: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "sequencer_msg",
			Help:      "number of sequencer msg actions",
		}),

		oracle: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "oracle",
			Help:      "number of oracle actions",
		}),

		deploy: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "deploy",
			Help:      "number of deploy actions",
		}),

		transact: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "transact",
			Help:      "number of transact actions",
		}),
	}
	r := prometheus.NewRegistry()
	errs := wrappers.Errs{}
	errs.Add(
		r.Register(m.createAsset),
		r.Register(m.mintAsset),
		r.Register(m.burnAsset),

		r.Register(m.transfer),

		r.Register(m.sequencerMsg),

		r.Register(m.oracle),

		r.Register(m.deploy),
		r.Register(m.transact),
		gatherer.Register(consts.Name, r),
	)
	return m, errs.Err
}
