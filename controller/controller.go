// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/AnomalyFi/hypersdk/builder"
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/gossiper"
	"github.com/AnomalyFi/hypersdk/network"
	hrpc "github.com/AnomalyFi/hypersdk/rpc"
	hstorage "github.com/AnomalyFi/hypersdk/storage"
	"github.com/AnomalyFi/hypersdk/vm"
	ametrics "github.com/ava-labs/avalanchego/api/metrics"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"go.uber.org/zap"

	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/config"
	"github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/genesis"
	"github.com/AnomalyFi/nodekit-seq/orderbook"
	"github.com/AnomalyFi/nodekit-seq/rpc"
	serverless "github.com/AnomalyFi/nodekit-seq/server-less"
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

	metrics *metrics

	metaDB database.Database

	orderBook *orderbook.OrderBook

	relayManager *RelayManager

	wsServer *rpc.WebSocketServer

	serverLess *serverless.ServerLess
}

func New() *vm.VM {
	return vm.New(&Controller{}, version.Version)
}

func (c *Controller) Initialize(
	inner *vm.VM,
	snowCtx *snow.Context,
	networkManager *network.Manager,
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
	jsonRPCHandler, err := hrpc.NewJSONRPCHandler(
		consts.Name,
		rpc.NewJSONRPCServer(c),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	apis[rpc.JSONRPCEndpoint] = jsonRPCHandler
	// websocket server for serving commitment callbacks esp for relayers
	wsServer, pubsubServer := rpc.NewWebSocketServer(c, c.config.GetStreamingBacklogSize())
	c.wsServer = wsServer
	apis[rpc.WSEndPoint] = pubsubServer

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
	// initiate serverless
	c.serverLess = serverless.NewServerLess(1024, 1024, snowCtx.Log)
	// initiate relayManager & relayHandler
	relayHandler, relaySender := networkManager.Register()
	c.relayManager = NewRelayManager(c.inner, c.serverLess, snowCtx)
	networkManager.SetHandler(relayHandler, NewRelayHandler(c))

	go c.serverLess.Serverless(c.relayManager, c.config.ServerlessPort)
	go c.relayManager.Run(relaySender)
	// Initialize order book used to track all open orders
	c.orderBook = orderbook.New(c, c.config.TrackedPairs, c.config.MaxOrdersPerPair)
	return c.config, c.genesis, build, gossip, blockDB, stateDB, apis, consts.ActionRegistry, consts.AuthRegistry, auth.Engines(), nil
}

func (c *Controller) Rules(t int64) chain.Rules {
	// TODO: extend with [UpgradeBytes]
	return c.genesis.Rules(t, c.snowCtx.NetworkID, c.snowCtx.ChainID, c.config.VerificationKey, c.config.GnarkPrecompileDecoderABI)
}

func (c *Controller) StateManager() chain.StateManager {
	return c.stateManager
}

func (c *Controller) Accepted(ctx context.Context, blk *chain.StatelessBlock) error {
	batch := c.metaDB.NewBatch()
	defer batch.Reset()

	// filter the transactions in websocket_packer.go and send to the listeners
	if err := c.wsServer.AcceptBlockWithSEQWasmTxs(blk); err != nil {
		c.Logger().Info("failed to send block to websocket server", zap.Error(err))
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
				result.Consumed,
				result.Fee,
			)
			if err != nil {
				return err
			}
		}
		if result.Success {
			switch action := tx.Action.(type) {
			case *actions.CreateAsset:
				c.metrics.createAsset.Inc()
			case *actions.MintAsset:
				c.metrics.mintAsset.Inc()
			case *actions.BurnAsset:
				c.metrics.burnAsset.Inc()
			case *actions.Transfer:
				c.metrics.transfer.Inc()
			case *actions.CreateOrder:
				c.metrics.createOrder.Inc()
				c.orderBook.Add(tx.ID(), tx.Auth.Actor(), action)
			case *actions.FillOrder:
				c.metrics.fillOrder.Inc()
				orderResult, err := actions.UnmarshalOrderResult(result.Output)
				if err != nil {
					// This should never happen
					return err
				}
				if orderResult.Remaining == 0 {
					c.orderBook.Remove(action.Order)
					continue
				}
				c.orderBook.UpdateRemaining(action.Order, orderResult.Remaining)
			case *actions.CloseOrder:
				c.metrics.closeOrder.Inc()
				c.orderBook.Remove(action.Order)
			case *actions.ImportAsset:
				c.metrics.importAsset.Inc()
			case *actions.ExportAsset:
				c.metrics.exportAsset.Inc()
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

func (c *Controller) SendRequestToAll(ctx context.Context, data []byte, relayerID int) error {
	return c.relayManager.SendRequestToAll(ctx, relayerID, data)
}

func (c *Controller) SendRequestToIndividual(ctx context.Context, data []byte, relayerID int, nodeID ids.NodeID) error {
	return c.relayManager.SendRequestToIndividual(ctx, relayerID, nodeID, data)
}
