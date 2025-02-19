// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"
	"fmt"
	"net/http"

	hactions "github.com/AnomalyFi/hypersdk/actions"

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
	rollupregistry "github.com/AnomalyFi/nodekit-seq/rollup_registry"
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

	jsonRPCServer  *rpc.JSONRPCServer
	archiver       *archiver.ORMArchiver
	rollupRegistry *rollupregistry.RollupRegistry

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

	if c.config.ArchiverConfig.ArchiverType == "sqlite" {
		c.config.ArchiverConfig.DSN = "/tmp/sqlite." + snowCtx.NodeID.String() + ".db"
		snowCtx.Log.Debug("setting archiver to", zap.String("dsn", c.config.ArchiverConfig.DSN))
	}
	// snowCtx.NodeID.String()
	c.archiver, err = archiver.NewORMArchiverFromConfig(&c.config.ArchiverConfig)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	c.rollupRegistry = rollupregistry.NewRollupRegistr()

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
	return c.genesis.Rules(t, c.snowCtx.NetworkID, c.snowCtx.ChainID, c.config.GetParsedWhiteListedAddress())
}

func (c *Controller) StateManager() chain.StateManager {
	return c.stateManager
}

func (c *Controller) UnitPrices(ctx context.Context) (fees.Dimensions, error) {
	return c.inner.UnitPrices(ctx)
}

func (c *Controller) NameSpacesPrice(ctx context.Context, namespaces []string) ([]uint64, error) {
	return c.inner.NameSpacesPrice(ctx, namespaces)
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

	go func() {
		err := c.archiver.InsertBlock(blk)
		if err != nil {
			c.Logger().Debug("err inserting block", zap.Error(err))
		}
	}()

	rollups := make([]*hactions.RollupInfo, 0)
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
				switch act.(type) { //nolint:gocritic,gosimple
				case *actions.Transfer:
					c.metrics.transfer.Inc()
				case *actions.SequencerMsg:
					c.metrics.sequencerMsg.Inc()
				case *actions.Auction:
					c.metrics.auction.Inc()
				case *actions.RollupRegistration:
					reg := act.(*actions.RollupRegistration) //nolint:gosimple
					rollups = append(rollups, &reg.Info)
					c.metrics.rollupRegister.Inc()
				case *actions.EpochExit:
					c.metrics.epochExit.Inc()
				}
			}
		}
	}
	currentEpoch := blk.Hght / uint64(c.inner.Rules(blk.Tmstmp).GetEpochLength())
	c.rollupRegistry.Update(currentEpoch, rollups)
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
