// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/AnomalyFi/hypersdk/builder"
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/fees"
	"github.com/AnomalyFi/hypersdk/gossiper"
	hrpc "github.com/AnomalyFi/hypersdk/rpc"
	hstorage "github.com/AnomalyFi/hypersdk/storage"
	"github.com/AnomalyFi/hypersdk/vm"
	ametrics "github.com/ava-labs/avalanchego/api/metrics"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/snow"
	"go.uber.org/zap"

	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/archiver"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/config"
	"github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/genesis"

	// "github.com/AnomalyFi/nodekit-seq/orderbook"

	"github.com/AnomalyFi/nodekit-seq/rpc"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/AnomalyFi/nodekit-seq/version"
)

var _ vm.Controller = (*Controller)(nil)

type Controller struct {
	inner *vm.VM

	snowCtx      *snow.Context
	genesis      *genesis.Genesis
	config       *config.Config
	stateManager *StateManager

	jsonRPCServer *rpc.JSONRPCServer
	archiver      *archiver.ORMArchiver

	metrics *metrics

	metaDB database.Database
}

func New() *vm.VM {
	return vm.New(&Controller{}, version.Version)
}

func (c *Controller) Initialize(
	inner *vm.VM,
	snowCtx *snow.Context,
	gatherer ametrics.MultiGatherer,
	genesisBytes []byte,
	upgradeBytes []byte, // subnets to allow for AWM
	configBytes []byte,
) (
	vm.Config,
	vm.Genesis,
	builder.Builder,
	gossiper.Gossiper,
	database.Database,
	database.Database,
	vm.Handlers,
	chain.ActionRegistry,
	chain.AuthRegistry,
	map[uint8]vm.AuthEngine,
	error,
) {
	c.inner = inner
	c.snowCtx = snowCtx
	c.stateManager = &StateManager{}

	// Instantiate metrics
	var err error
	c.metrics, err = newMetrics(gatherer)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	// Load config and genesis
	c.config, err = config.New(c.snowCtx.NodeID, configBytes)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	c.snowCtx.Log.SetLevel(c.config.GetLogLevel())
	snowCtx.Log.Info("initialized config", zap.Bool("loaded", c.config.Loaded()), zap.Any("contents", c.config))

	c.genesis, err = genesis.New(genesisBytes, upgradeBytes)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf(
			"unable to read genesis: %w",
			err,
		)
	}
	snowCtx.Log.Info("loaded genesis", zap.Any("genesis", c.genesis))

	// Create DBs
	blockDB, stateDB, metaDB, err := hstorage.New(snowCtx.ChainDataDir, gatherer)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	c.metaDB = metaDB

	// Create handlers
	//
	// hypersdk handler are initiatlized automatically, you just need to
	// initialize custom handlers here.
	apis := map[string]http.Handler{}
	jsonRPCServer := rpc.NewJSONRPCServer(c)
	c.jsonRPCServer = jsonRPCServer
	jsonRPCHandler, err := hrpc.NewJSONRPCHandler(
		consts.Name,
		jsonRPCServer,
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	c.archiver, err = archiver.NewORMArchiverFromConfig(&c.config.ArchiverConfig)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	apis[rpc.JSONRPCEndpoint] = jsonRPCHandler

	// Create builder and gossiper
	var (
		build  builder.Builder
		gossip gossiper.Gossiper
	)
	if c.config.TestMode {
		c.inner.Logger().Info("running build and gossip in test mode")
		build = builder.NewManual(inner)
		gossip = gossiper.NewManual(inner)
	} else {
		build = builder.NewTime(inner)
		gcfg := gossiper.DefaultProposerConfig()
		gcfg.GossipMaxSize = c.config.GossipMaxSize
		gcfg.GossipProposerDiff = c.config.GossipProposerDiff
		gcfg.GossipProposerDepth = c.config.GossipProposerDepth
		gcfg.NoGossipBuilderDiff = c.config.NoGossipBuilderDiff
		gcfg.VerifyTimeout = c.config.VerifyTimeout
		gossip, err = gossiper.NewProposer(inner, gcfg)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
	}

	return c.config, c.genesis, build, gossip, blockDB, stateDB, apis, consts.ActionRegistry, consts.AuthRegistry, auth.Engines(), nil
}

func (c *Controller) Rules(t int64) chain.Rules {
	// TODO: extend with [UpgradeBytes]
	return c.genesis.Rules(t, c.snowCtx.NetworkID, c.snowCtx.ChainID)
}

func (c *Controller) StateManager() chain.StateManager {
	return c.stateManager
}

func (c *Controller) UnitPrices(ctx context.Context) (fees.Dimensions, error) {
	return c.inner.UnitPrices(ctx)
}

func (c *Controller) Submit(
	ctx context.Context,
	verifySig bool,
	txs []*chain.Transaction,
) (errs []error) {
	return c.inner.Submit(ctx, verifySig, txs)
}

// TODO I can add the blocks to the JSON RPC Server here instead of REST API
func (c *Controller) Accepted(ctx context.Context, blk *chain.StatelessBlock) error {
	batch := c.metaDB.NewBatch()
	defer batch.Reset()

	go c.archiver.InsertBlock(blk)

	if err := c.jsonRPCServer.AcceptBlock(blk); err != nil {
		c.inner.Logger().Fatal("unable to accept block in json-rpc server", zap.Error(err))
	}

	results := blk.Results()
	for i, tx := range blk.Txs {
		result := results[i]
		if c.config.GetStoreTransactions() {
			err := storage.StoreTransaction(
				ctx,
				batch,
				tx.ID(),
				blk.GetTimestamp(),
				result.Success,
				result.Units,
				result.Fee,
			)
			if err != nil {
				return err
			}
		}
		if result.Success {
			for _, act := range tx.Actions {
				switch act.(type) {
				case *actions.CreateAsset:
					c.metrics.createAsset.Inc()
				case *actions.MintAsset:
					c.metrics.mintAsset.Inc()
				case *actions.BurnAsset:
					c.metrics.burnAsset.Inc()
				case *actions.Transfer:
					c.metrics.transfer.Inc()
				case *actions.SequencerMsg:
					c.metrics.sequencerMsg.Inc()
				}
			}
		}
	}
	return batch.Write()
}

func (*Controller) Rejected(context.Context, *chain.StatelessBlock) error {
	return nil
}

func (*Controller) Shutdown(context.Context) error {
	// Do not close any databases provided during initialization. The VM will
	// close any databases your provided.
	return nil
}
