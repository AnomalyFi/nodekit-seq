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
	transfer prometheus.Counter

	sequencerMsg prometheus.Counter
	auction      prometheus.Counter

	rollupRegister prometheus.Counter
	epochExit      prometheus.Counter
}

func newMetrics(gatherer ametrics.MultiGatherer) (*metrics, error) {
	m := &metrics{
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

		auction: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "auction",
			Help:      "number of auction actions",
		}),

		rollupRegister: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "rollup_register",
			Help:      "number of rollup register actions",
		}),

		epochExit: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "actions",
			Name:      "epoch_exit",
			Help:      "number of epoch exit actions",
		}),
	}
	r := prometheus.NewRegistry()
	errs := wrappers.Errs{}
	errs.Add(
		r.Register(m.transfer),

		r.Register(m.sequencerMsg),
		r.Register(m.auction),

		r.Register(m.rollupRegister),
		r.Register(m.epochExit),

		gatherer.Register(consts.Name, r),
	)
	return m, errs.Err
}
