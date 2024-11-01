// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package e2e_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ava-labs/avalanche-network-runner/rpcpb"
	"github.com/ava-labs/avalanchego/config"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/fatih/color"
	"github.com/stretchr/testify/require"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	hconsts "github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/crypto/bls"
	"github.com/AnomalyFi/hypersdk/crypto/ed25519"
	"github.com/AnomalyFi/hypersdk/pubsub"
	"github.com/AnomalyFi/hypersdk/rpc"
	"github.com/AnomalyFi/nodekit-seq/actions"
	"github.com/AnomalyFi/nodekit-seq/auth"
	"github.com/AnomalyFi/nodekit-seq/consts"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/AnomalyFi/nodekit-seq/types"

	hutils "github.com/AnomalyFi/hypersdk/utils"
	trpc "github.com/AnomalyFi/nodekit-seq/rpc"
	runner_sdk "github.com/ava-labs/avalanche-network-runner/client"
	ginkgo "github.com/onsi/ginkgo/v2"
)

const (
	startAmount = uint64(10000000000000000000)
	sendAmount  = uint64(5000)

	healthPollInterval = 10 * time.Second
)

func TestE2e(t *testing.T) {
	ginkgo.RunSpecs(t, "seqvm e2e test suites")
}

var (
	requestTimeout time.Duration

	networkRunnerLogLevel      string
	avalanchegoLogLevel        string
	avalanchegoLogDisplayLevel string
	gRPCEp                     string
	gRPCGatewayEp              string

	execPath  string
	pluginDir string

	vmGenesisPath    string
	vmConfigPath     string
	vmConfig         string
	subnetConfigPath string
	outputPath       string

	mode string

	logsDir string

	blockchainID string

	trackSubnetsOpt runner_sdk.OpOption

	numValidators uint

	priorityFee uint64 = 0
)

func init() {
	flag.DurationVar(
		&requestTimeout,
		"request-timeout",
		150*time.Second,
		"timeout for transaction issuance and confirmation",
	)

	flag.StringVar(
		&networkRunnerLogLevel,
		"network-runner-log-level",
		"info",
		"gRPC server endpoint",
	)

	flag.StringVar(
		&avalanchegoLogLevel,
		"avalanchego-log-level",
		"info",
		"log level for avalanchego",
	)

	flag.StringVar(
		&avalanchegoLogDisplayLevel,
		"avalanchego-log-display-level",
		"error",
		"log display level for avalanchego",
	)

	flag.StringVar(
		&gRPCEp,
		"network-runner-grpc-endpoint",
		"0.0.0.0:8080",
		"gRPC server endpoint",
	)
	flag.StringVar(
		&gRPCGatewayEp,
		"network-runner-grpc-gateway-endpoint",
		"0.0.0.0:8081",
		"gRPC gateway endpoint",
	)

	flag.StringVar(
		&execPath,
		"avalanchego-path",
		"",
		"avalanchego executable path",
	)

	flag.StringVar(
		&pluginDir,
		"avalanchego-plugin-dir",
		"",
		"avalanchego plugin directory",
	)

	flag.StringVar(
		&vmGenesisPath,
		"vm-genesis-path",
		"",
		"VM genesis file path",
	)

	flag.StringVar(
		&vmConfigPath,
		"vm-config-path",
		"",
		"VM configfile path",
	)

	flag.StringVar(
		&subnetConfigPath,
		"subnet-config-path",
		"",
		"Subnet configfile path",
	)

	flag.StringVar(
		&outputPath,
		"output-path",
		"",
		"output YAML path to write local cluster information",
	)

	flag.StringVar(
		&mode,
		"mode",
		"test",
		"'test' to shut down cluster after tests, 'run' to skip tests and only run without shutdown",
	)

	flag.UintVar(
		&numValidators,
		"num-validators",
		5,
		"number of validators per blockchain",
	)
}

const (
	modeTest     = "test"
	modeFullTest = "full-test" // runs state sync
	modeRun      = "run"
)

var anrCli runner_sdk.Client

var _ = ginkgo.BeforeSuite(func() {
	require := require.New(ginkgo.GinkgoT())

	require.Contains([]string{modeTest, modeFullTest, modeRun}, mode)
	require.Greater(numValidators, uint(0))
	logLevel, err := logging.ToLevel(networkRunnerLogLevel)
	require.NoError(err)
	logFactory := logging.NewFactory(logging.Config{
		DisplayLevel: logLevel,
		LogLevel:     logLevel,
	})
	log, err := logFactory.Make("main")
	require.NoError(err)

	anrCli, err = runner_sdk.New(runner_sdk.Config{
		Endpoint:    gRPCEp,
		DialTimeout: 10 * time.Second,
	}, log)
	require.NoError(err)

	hutils.Outf(
		"{{green}}sending 'start' with binary path:{{/}} %q (%q)\n",
		execPath,
		consts.ID,
	)

	// Load config data
	if len(vmConfigPath) > 0 {
		configData, err := os.ReadFile(vmConfigPath)
		require.NoError(err)
		vmConfig = string(configData)
	} else {
		vmConfig = "{}"
	}

	// Start cluster
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	resp, err := anrCli.Start(
		ctx,
		execPath,
		runner_sdk.WithPluginDir(pluginDir),
		runner_sdk.WithGlobalNodeConfig(fmt.Sprintf(`{
				"log-level":"%s",
				"log-display-level":"%s",
				"proposervm-use-current-height":true,
				"throttler-inbound-validator-alloc-size":"10737418240",
				"throttler-inbound-at-large-alloc-size":"10737418240",
				"throttler-inbound-node-max-at-large-bytes":"10737418240",
				"throttler-inbound-node-max-processing-msgs":"1000000",
				"throttler-inbound-bandwidth-refill-rate":"1073741824",
				"throttler-inbound-bandwidth-max-burst-size":"1073741824",
				"throttler-inbound-cpu-validator-alloc":"100000",
				"throttler-inbound-cpu-max-non-validator-usage":"100000",
				"throttler-inbound-cpu-max-non-validator-node-usage":"100000",
				"throttler-inbound-disk-validator-alloc":"10737418240000",
				"throttler-outbound-validator-alloc-size":"10737418240",
				"throttler-outbound-at-large-alloc-size":"10737418240",
				"throttler-outbound-node-max-at-large-bytes": "10737418240",
				"network-compression-type":"zstd",
				"consensus-app-concurrency":"512",
				"profile-continuous-enabled":true,
				"profile-continuous-freq":"1m",
				"http-host":"",
				"http-allowed-origins": "*",
				"http-allowed-hosts": "*"
			}`,
			avalanchegoLogLevel,
			avalanchegoLogDisplayLevel,
		)),
	)
	cancel()
	require.NoError(err)
	hutils.Outf(
		"{{green}}successfully started cluster:{{/}} %s {{green}}subnets:{{/}} %+v\n",
		resp.ClusterInfo.RootDataDir,
		resp.GetClusterInfo().GetSubnets(),
	)
	logsDir = resp.GetClusterInfo().GetRootDataDir()

	// Add 5 validators (already have BLS key registered)
	subnet := []string{}
	for i := 1; i <= int(numValidators); i++ {
		n := fmt.Sprintf("node%d", i)
		subnet = append(subnet, n)
	}
	specs := []*rpcpb.BlockchainSpec{
		{
			VmName:      consts.Name,
			Genesis:     vmGenesisPath,
			ChainConfig: vmConfigPath,
			SubnetSpec: &rpcpb.SubnetSpec{
				SubnetConfig: subnetConfigPath,
				Participants: subnet,
			},
		},
	}

	// Create subnet
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
	sresp, err := anrCli.CreateBlockchains(
		ctx,
		specs,
	)
	cancel()
	require.NoError(err)

	blockchainID = sresp.ChainIds[0]
	subnetID := sresp.ClusterInfo.CustomChains[blockchainID].SubnetId
	hutils.Outf(
		"{{green}}successfully added chain:{{/}} %s {{green}}subnet:{{/}} %s {{green}}participants:{{/}} %+v\n",
		blockchainID,
		subnetID,
		subnet,
	)
	trackSubnetsOpt = runner_sdk.WithGlobalNodeConfig(fmt.Sprintf(`{"%s":"%s"}`,
		config.TrackSubnetsKey,
		subnetID,
	))

	require.NotEmpty(blockchainID)
	require.NotEmpty(logsDir)

	cctx, ccancel := context.WithTimeout(context.Background(), 2*time.Minute)
	status, err := anrCli.Status(cctx)
	ccancel()
	require.NoError(err)
	nodeInfos := status.GetClusterInfo().GetNodeInfos()
	privBytes, err := codec.LoadHex(
		"323b1d8f4eed5f0da9da93071b034f2dce9d2d22692c172f3cb252a64ddfafd01b057de320297c29ad0c1f589ea216869cf1938d88c9fbd70d6748323dbf2fa7", //nolint:lll
		ed25519.PrivateKeyLen,
	)
	require.NoError(err)
	instances = []instance{}
	for _, nodeName := range subnet {
		info := nodeInfos[nodeName]
		u := fmt.Sprintf("%s/ext/bc/%s", info.Uri, blockchainID)
		bid, err := ids.FromString(blockchainID)
		require.NoError(err)
		nodeID, err := ids.NodeIDFromString(info.GetId())
		require.NoError(err)
		cli := rpc.NewJSONRPCClient(u)
		wsCli, err := rpc.NewWebSocketClient(u, rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
		require.NoError(err)
		// After returning healthy, the node may not respond right away
		//
		// TODO: figure out why
		var networkID uint32
		for i := 0; i < 10; i++ {
			networkID, _, _, err = cli.Network(context.TODO())
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
		}
		require.NoError(err)

		instances = append(instances, instance{
			nodeID:   nodeID,
			uri:      u,
			cli:      cli,
			tcli:     trpc.NewJSONRPCClient(u, networkID, bid),
			wsCli:    wsCli,
			multicli: trpc.NewMultiJSONRPCClientWithED25519Factory(u, networkID, bid, privBytes),
		})
	}

	// Load default pk
	priv = ed25519.PrivateKey(privBytes)
	factory = auth.NewED25519Factory(priv)
	rsender = auth.NewED25519Address(priv.PublicKey())
	sender = codec.MustAddressBech32(consts.HRP, rsender)
	hutils.Outf("\n{{yellow}}$ loaded address:{{/}} %s\n\n", sender)
})

var (
	priv    ed25519.PrivateKey
	factory *auth.ED25519Factory
	rsender codec.Address
	sender  string

	instances []instance
)

type instance struct {
	nodeID   ids.NodeID
	uri      string
	cli      *rpc.JSONRPCClient
	tcli     *trpc.JSONRPCClient
	wsCli    *rpc.WebSocketClient
	multicli *trpc.MultiJSONRPCClient
}

var _ = ginkgo.AfterSuite(func() {
	require := require.New(ginkgo.GinkgoT())

	switch mode {
	case modeTest, modeFullTest:
		hutils.Outf("{{red}}shutting down cluster{{/}}\n")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		_, err := anrCli.Stop(ctx)
		cancel()
		require.NoError(err)

	case modeRun:
		hutils.Outf("{{yellow}}skipping cluster shutdown{{/}}\n\n")
		hutils.Outf("{{cyan}}Blockchain:{{/}} %s\n", blockchainID)
		for _, member := range instances {
			hutils.Outf("%s URI: %s\n", member.nodeID, member.uri)
		}
	}
	require.NoError(anrCli.Close())
})

var _ = ginkgo.Describe("[Ping]", func() {
	require := require.New(ginkgo.GinkgoT())

	ginkgo.It("can ping", func() {
		for _, inst := range instances {
			cli := inst.cli
			ok, err := cli.Ping(context.Background())
			require.NoError(err)
			require.True(ok)
		}
	})
})

var _ = ginkgo.Describe("[Network]", func() {
	require := require.New(ginkgo.GinkgoT())

	ginkgo.It("can get network", func() {
		for _, inst := range instances {
			cli := inst.cli
			_, _, chainID, err := cli.Network(context.Background())
			require.NoError(err)
			require.NotEqual(chainID, ids.Empty)
		}
	})
})

var _ = ginkgo.Describe("[Test]", func() {
	require := require.New(ginkgo.GinkgoT())

	switch mode {
	case modeRun:
		hutils.Outf("{{yellow}}skipping tests{{/}}\n")
		return
	}

	ginkgo.It("transfer in a single node (raw)", func() {
		nativeBalance, err := instances[0].tcli.Balance(context.TODO(), sender)
		require.NoError(err)
		require.Equal(nativeBalance, startAmount)

		other, err := ed25519.GeneratePrivateKey()
		require.NoError(err)
		aother := auth.NewED25519Address(other.PublicKey())

		ginkgo.By("issue Transfer to the first node", func() {
			// Generate transaction
			parser, err := instances[0].tcli.Parser(context.TODO())
			require.NoError(err)
			submit, tx, maxFee, err := instances[0].cli.GenerateTransaction(
				context.Background(),
				parser,
				[]chain.Action{&actions.Transfer{
					To:    aother,
					Value: sendAmount,
				}},
				factory,
				priorityFee,
			)
			require.NoError(err)
			hutils.Outf("{{yellow}}generated transaction{{/}}\n")

			// Broadcast and wait for transaction
			require.NoError(submit(context.Background()))
			hutils.Outf("{{yellow}}submitted transaction{{/}}\n")
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			success, fee, err := instances[0].tcli.WaitForTransaction(ctx, tx.ID())
			cancel()
			require.NoError(err)
			require.True(success)
			hutils.Outf("{{yellow}}found transaction{{/}}\n")

			// Check sender balance
			balance, err := instances[0].tcli.Balance(context.Background(), sender)
			require.NoError(err)
			hutils.Outf(
				"{{yellow}}start=%d fee=%d send=%d balance=%d{{/}}\n",
				startAmount,
				maxFee,
				sendAmount,
				balance,
			)
			require.Equal(balance, startAmount-fee-sendAmount)
			hutils.Outf("{{yellow}}fetched balance{{/}}\n")
		})

		ginkgo.By("check if Transfer has been accepted from all nodes", func() {
			for _, inst := range instances {
				color.Blue("checking %q", inst.uri)

				// Ensure all blocks processed
				for {
					_, h, _, err := inst.cli.Accepted(context.Background())
					require.NoError(err)
					if h > 0 {
						break
					}
					time.Sleep(1 * time.Second)
				}

				// Check balance of recipient
				balance, err := inst.tcli.Balance(context.Background(), codec.MustAddressBech32(consts.HRP, aother))
				require.NoError(err)
				require.Equal(balance, sendAmount)
			}
		})
	})

	ginkgo.It("submit a sequencer msg", func() {
		blk := new(chain.StatefulBlock)
		results := make([]*chain.Result, 1)

		ginkgo.By("issue SequencerMsg to the first node", func() {
			parser, err := instances[0].tcli.Parser(context.TODO())
			require.NoError(err)
			chainID := 45200
			chainIDBytes := make([]byte, 8)
			binary.LittleEndian.PutUint64(chainIDBytes, uint64(chainID))
			data := make([]byte, 200)
			_, err = rand.Read(data)
			require.NoError(err)

			submit, tx, maxFee, err := instances[0].cli.GenerateTransaction(
				context.Background(),
				parser,
				[]chain.Action{&actions.SequencerMsg{
					FromAddress: rsender,
					Data:        data,
					ChainID:     chainIDBytes,
					RelayerID:   0,
				}},
				factory,
				priorityFee,
			)
			require.NoError(err)
			hutils.Outf("{{yellow}}generated sequencer message transaction{{/}}\n")
			require.NoError(submit(context.Background()))
			hutils.Outf("{{yellow}}submitted transaction{{/}}\n")
			// listen block
			wsCli, err := rpc.NewWebSocketClient(instances[0].uri, rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
			require.NoError(err)
			err = wsCli.RegisterBlocks()
			require.NoError(err)
			found := false
			waitTillIncluded := func(ctx context.Context) error {
				for {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if found {
							return nil
						}

						b, rs, _, _, err := wsCli.ListenBlock(context.Background(), parser)
						require.NoError(err)
						for _, t := range b.Txs {
							if t.ID() == tx.ID() {
								found = true
								hutils.Outf("{{green}}inclusion block found{{/}}\n")
								break
							}
						}

						blk = b
						results = rs
					}
				}
			}

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			err = waitTillIncluded(ctx)
			require.NoError(err)
			success, _, err := instances[0].tcli.WaitForTransaction(ctx, tx.ID())
			cancel()
			require.NoError(err)
			require.True(success)
			hutils.Outf("{{yellow}}found transaction{{/}}\n")
			hutils.Outf(
				"{{yellow}}chainID=%d fee=%d sender=%s{{/}}\n",
				chainID,
				maxFee,
				sender,
			)

			// check if NMT Proof is correct
			nID := tx.Actions[0].NMTNamespace()
			require.Equal(chainIDBytes, nID)
			root := blk.NMTRoot
			require.Equal(hconsts.NMTRootLen, len(root))
			proof, ok := blk.NMTProofs[hex.EncodeToString(nID)]
			require.True(ok)
			hutils.Outf("{{green}}proof:{{/}}%+v \n", proof)
			hutils.Outf("{{green}}proofs:{{/}}%+v \n", blk.NMTProofs)
			actionData := make([]byte, 0)
			// should only contain results of the sequencer msg tx
			require.Equal(1, len(results))
			// prepare action data to be verified
			txID := tx.ID()
			txResult := results[0]
			actionData = append(actionData, nID...)
			actionData = append(actionData, txID[:]...)
			// append `result` of Action 0 of Tx 0
			for k := 0; k < len(txResult.Outputs[0]); k++ {
				actionData = append(actionData, txResult.Outputs[0][k]...)
			}

			hutils.Outf("{{green}}actionData:{{/}}%+v \n", hex.EncodeToString(actionData))

			// leaves without prefix namespace ID
			leaves := make([][]byte, 0, 1)
			leaves = append(leaves, actionData)

			verified := proof.VerifyNamespace(sha256.New(), nID, leaves, root)
			require.True(verified)
		})
	})

	ginkgo.It("ensure SubmitMsgTx work", func() {
		ginkgo.By("issuing some transactions", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			tpriv, err := ed25519.GeneratePrivateKey()
			require.NoError(err)
			rsender := auth.NewED25519Address(tpriv.PublicKey())
			txActions := []chain.Action{&actions.SequencerMsg{
				ChainID:     []byte("nkit"),
				Data:        []byte("somedata"),
				FromAddress: rsender,
				RelayerID:   0,
			}, &actions.SequencerMsg{
				ChainID:     []byte("nkit"),
				Data:        []byte("somedata2"),
				FromAddress: rsender,
				RelayerID:   0,
			}}
			txID, err := instances[0].multicli.GenerateAndSubmitTx(ctx, txActions, 0)
			require.NoError(err)
			hutils.Outf("{{green}}txID of submitted data:{{/}}%s \n", txID.String())
			require.NoError(err)
			success, _, err := instances[0].tcli.WaitForTransaction(ctx, txID)
			require.NoError(err)
			require.True(success)
		})
	})

	ginkgo.It("test rpc server and archiver is working correctly", func() {
		var blk *chain.StatefulBlock
		var blkID ids.ID
		// var txID ids.ID
		var txIDn ids.ID
		namespace := hex.EncodeToString([]byte("nkit"))
		// util funcs
		parser, err := instances[0].tcli.Parser(context.TODO())
		require.NoError(err)
		wsCli, err := rpc.NewWebSocketClient(instances[0].uri, rpc.DefaultHandshakeTimeout, pubsub.MaxPendingMessages, pubsub.MaxReadMessageSize)
		require.NoError(err)
		err = wsCli.RegisterBlocks()
		require.NoError(err)
		waitTillIncluded := func(ctx context.Context, txIDStr string, submitFunc func(context.Context) error) (bool, *chain.StatefulBlock, error) {
			submitted := false
			txID, err := ids.FromString(txIDStr)
			require.NoError(err)
			for {
				select {
				case <-ctx.Done():
					return false, nil, ctx.Err()
				default:
					b, _, _, _, err := wsCli.ListenBlock(context.Background(), parser)
					require.NoError(err)
					for _, t := range b.Txs {
						if t.ID() == txID {
							hutils.Outf("{{green}}inclusion block found{{/}}\n")
							return true, b, nil
						}
					}

					if !submitted {
						err := submitFunc(ctx)
						if err != nil {
							hutils.Outf("{{red}}failed to submit block: %s{{/}}\n", err)
						} else {
							hutils.Outf("{{green}}submitted tx: %s{{/}}\n", txIDStr)
							submitted = true
						}
					}
				}
			}
		}
		ginkgo.By("issuing some transactions", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			tpriv, err := ed25519.GeneratePrivateKey()
			require.NoError(err)
			rsender := auth.NewED25519Address(tpriv.PublicKey())
			txActions := []chain.Action{&actions.SequencerMsg{
				ChainID:     []byte("nkit"),
				Data:        []byte("somedata"),
				FromAddress: rsender,
				RelayerID:   0,
			}, &actions.SequencerMsg{
				ChainID:     []byte("nkit"),
				Data:        []byte("somedata2"),
				FromAddress: rsender,
				RelayerID:   0,
			}}
			parser, err := instances[0].tcli.Parser(ctx)
			require.NoError(err)
			submit, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
			require.NoError(err)
			txID := tx.ID()
			included, block, err := waitTillIncluded(ctx, txID.String(), submit)
			require.True(included)
			require.NoError(err)
			require.True(included)
			require.NotNil(block)
			blockID, err := block.ID()
			require.NoError(err)
			txIDn = txID
			blkID = blockID
			blk = block
		})

		// wait for more blocks to be produced
		time.Sleep(10 * time.Second)

		ginkgo.By("issuing GetBlockTransactions", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			resp, err := instances[0].tcli.GetBlockTransactions(ctx, blkID.String())
			require.NoError(err)
			require.Equal(2, len(resp.Txs))

			hutils.Outf("{{green}}height: %d blockID wanted:{{/}}%s \n", blk.Hght, blkID.String())
			txInBlock := resp.Txs[0]
			require.Equal(txIDn.String(), txInBlock.TxID)
		})

		ginkgo.By("issuing GetBlockTransactions", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			resp, err := instances[0].tcli.GetBlockTransactionsByNamespace(ctx, blk.Hght, namespace)
			require.NoError(err)
			require.Equal(2, len(resp.Txs))

			resp, err = instances[0].tcli.GetBlockTransactionsByNamespace(ctx, blk.Hght, "someothernamespace")
			require.NoError(err)
			require.Equal(0, len(resp.Txs))
		})

		ginkgo.By("issuing GetBlockHeadersByHeight", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			resp, err := instances[0].tcli.GetBlockHeadersByHeight(ctx, blk.Hght, blk.Tmstmp+5000)
			require.NoError(err)
			require.Greater(len(resp.Blocks), 0)

			found := false
			hutils.Outf("{{green}}height: %d blockID wanted:{{/}}%s \n", blk.Hght, blkID.String())
			for _, b := range resp.Blocks {
				hutils.Outf("{{green}}height: %d blockID:{{/}}%s \n", b.Height, b.BlockID)
				if b.BlockID == blkID.String() {
					found = true
				}
			}
			fmt.Printf("prev: %+v, blocks[0]: %+v, next: %+v, blocks[-1]: %+v\n", resp.Prev, resp.Blocks[0], resp.Next, resp.Blocks[len(resp.Blocks)-1])
			require.True(found)
			require.NotEqual(resp.Next, types.BlockInfo{})
			require.Equal(resp.Next.Height-1, resp.Blocks[len(resp.Blocks)-1].Height)
			require.Equal(resp.Prev.Height+1, resp.Blocks[0].Height)
		})

		ginkgo.By("issuing GetBlockHeadersByID", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			resp, err := instances[0].tcli.GetBlockHeadersID(ctx, blkID.String(), blk.Tmstmp+5000)
			require.NoError(err)
			require.Greater(len(resp.Blocks), 0)

			found := false
			blkID, err := blk.ID()
			require.NoError(err)
			for _, b := range resp.Blocks {
				if b.BlockID == blkID.String() {
					found = true
				}
			}
			fmt.Printf("prev: %+v, blocks[0]: %+v, next: %+v, blocks[-1]: %+v\n", resp.Prev, resp.Blocks[0], resp.Next, resp.Blocks[len(resp.Blocks)-1])
			require.True(found)
			require.Equal(resp.Next.Height-1, resp.Blocks[len(resp.Blocks)-1].Height)
			require.Equal(resp.Prev.Height+1, resp.Blocks[0].Height)
		})

		ginkgo.By("issuing GetBlockHeadersByStart", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			resp, err := instances[0].tcli.GetBlockHeadersByStartTimeStamp(ctx, blk.Tmstmp, blk.Tmstmp+5000)
			require.NoError(err)
			require.Greater(len(resp.Blocks), 0)

			found := false
			blkID, err := blk.ID()
			require.NoError(err)
			for _, b := range resp.Blocks {
				if b.BlockID == blkID.String() {
					found = true
				}
			}
			fmt.Printf("prev: %+v, blocks[0]: %+v, next: %+v, blocks[-1]: %+v\n", resp.Prev, resp.Blocks[0], resp.Next, resp.Blocks[len(resp.Blocks)-1])
			require.True(found)
			require.Equal(resp.Next.Height-1, resp.Blocks[len(resp.Blocks)-1].Height)
			require.Equal(resp.Prev.Height+1, resp.Blocks[0].Height)
		})

		ginkgo.By("issuing GetCommitmentBlocks", func() {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			resp, err := instances[0].tcli.GetCommitmentBlocks(ctx, blk.Hght-1, blk.Hght, 100)
			require.NoError(err)
			require.Equal(len(resp.Blocks), 2)
		})

	})

	// TODO: This is a flaky test. It passes only if the transaction is included just in right blocks. If not tests fails.
	ginkgo.It("test auction is working perfectly", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		tpriv, err := bls.GeneratePrivateKey()
		require.NoError(err)
		pubKey := bls.PublicFromPrivateKey(tpriv)
		pubKeyBytes := bls.PublicKeyToBytes(pubKey)
		builderSEQAddress := auth.NewBLSAddress(pubKey)
		_, hght, _, err := instances[0].cli.Accepted(ctx)
		require.NoError(err)
		if hght%6 != 0 {
			_, hght, _, err = instances[0].cli.Accepted(ctx)
			require.NoError(err)
			time.Sleep(50 * time.Millisecond)
		}
		epochNumber, err := instances[0].cli.GetCurrentEpoch()
		require.NoError(err)
		msg := make([]byte, 16)
		binary.BigEndian.PutUint64(msg[:8], epochNumber+1)
		binary.BigEndian.PutUint64(msg[8:], 100)
		msg = append(msg, pubKeyBytes...)
		sig := bls.Sign(msg, tpriv)
		txActions := []chain.Action{
			&actions.Transfer{
				To:    builderSEQAddress,
				Value: 200,
			},
			&actions.Auction{
				AuctionInfo: actions.AuctionInfo{
					EpochNumber:       epochNumber + 1,
					BidPrice:          100,
					BuilderSEQAddress: builderSEQAddress,
				},
				BuilderPublicKey: pubKeyBytes,
				BuilderSignature: bls.SignatureToBytes(sig),
			},
		}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s at height:{{/}}\n", tx.ID().String(), hght)

		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.True(result.Success)

		if result.Success {
			pubKeyBytesFromCli, err := instances[0].tcli.GetBuilder(context.Background(), epochNumber+1)
			require.NoError(err)
			require.Equal(pubKeyBytes, *pubKeyBytesFromCli)
		}
	})

	ginkgo.It("issuing invalid auction transaction: epoch not equal to next epoch", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		tpriv, err := bls.GeneratePrivateKey()
		require.NoError(err)
		pubKey := bls.PublicFromPrivateKey(tpriv)
		pubKeyBytes := bls.PublicKeyToBytes(pubKey)
		builderSEQAddress := auth.NewBLSAddress(pubKey)
		currEpoch, err := instances[0].cli.GetCurrentEpoch()
		require.NoError(err)
		msg := make([]byte, 16)
		binary.BigEndian.PutUint64(msg[:8], currEpoch+1)
		binary.BigEndian.PutUint64(msg[8:], 100)
		msg = append(msg, pubKeyBytes...)
		sig := bls.Sign(msg, tpriv)
		txActions := []chain.Action{
			&actions.Transfer{
				To:    builderSEQAddress,
				Value: 200,
			},
			&actions.Auction{
				AuctionInfo: actions.AuctionInfo{
					EpochNumber:       currEpoch,
					BidPrice:          100,
					BuilderSEQAddress: builderSEQAddress,
				},
				BuilderPublicKey: pubKeyBytes,
				BuilderSignature: bls.SignatureToBytes(sig),
			},
		}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())
		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.Contains(string(result.Error), "epoch number is not valid")
		require.False(result.Success)
	})

	// TODO: This is a flaky test. It passes only if the transaction is included just in right blocks. If not test fails.
	ginkgo.It("issuing invalid auction transaction: wrong signature", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		tpriv, err := bls.GeneratePrivateKey()
		require.NoError(err)
		pubKey := bls.PublicFromPrivateKey(tpriv)
		pubKeyBytes := bls.PublicKeyToBytes(pubKey)
		builderSEQAddress := auth.NewBLSAddress(pubKey)
		currEpoch, err := instances[0].cli.GetCurrentEpoch()
		require.NoError(err)
		msg := make([]byte, 16)
		binary.BigEndian.PutUint64(msg[:8], currEpoch+1)
		binary.BigEndian.PutUint64(msg[8:], 100)
		msg = append(msg, pubKeyBytes...)
		tpriv2, err := bls.GeneratePrivateKey()
		require.NoError(err)
		sig := bls.Sign(msg, tpriv2)
		txActions := []chain.Action{
			&actions.Transfer{
				To:    builderSEQAddress,
				Value: 200,
			},
			&actions.Auction{
				AuctionInfo: actions.AuctionInfo{
					EpochNumber:       currEpoch + 1,
					BidPrice:          100,
					BuilderSEQAddress: builderSEQAddress,
				},
				BuilderPublicKey: pubKeyBytes,
				BuilderSignature: bls.SignatureToBytes(sig),
			},
		}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())

		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.False(result.Success)
		require.Contains(string(result.Error), "invalid bidder signature")
	})

	ginkgo.It("test register rollup", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		currEpoch, err := instances[0].cli.GetCurrentEpoch()
		require.NoError(err)
		txActions := []chain.Action{&actions.RollupRegistration{
			Info: hactions.RollupInfo{
				Namespace:           []byte("nkit"),
				FeeRecipient:        rsender,
				AuthoritySEQAddress: rsender,
			},
			Namespace:  []byte("nkit"),
			StartEpoch: currEpoch + 5,
			OpCode:     actions.CreateRollup,
		}}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())

		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.True(result.Success)
	})
	ginkgo.It("test update rollup info", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		tpriv, err := bls.GeneratePrivateKey()
		require.NoError(err)

		pubKey := bls.PublicFromPrivateKey(tpriv)
		seqAddress := auth.NewBLSAddress(pubKey)
		txActions := []chain.Action{&actions.RollupRegistration{
			Info: hactions.RollupInfo{
				Namespace:           []byte("nkit"),
				FeeRecipient:        seqAddress,
				AuthoritySEQAddress: seqAddress,
			},
			Namespace: []byte("nkit"),
			OpCode:    actions.UpdateRollup,
		}}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())

		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.True(result.Success)
	})
	ginkgo.It("test delete rollup info", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		currEpoch, err := instances[0].cli.GetCurrentEpoch()
		require.NoError(err)
		txActions := []chain.Action{
			&actions.RollupRegistration{
				Info: hactions.RollupInfo{
					Namespace:           []byte("nkit2"),
					FeeRecipient:        rsender,
					AuthoritySEQAddress: rsender,
				},
				StartEpoch: currEpoch + 5,
				Namespace:  []byte("nkit2"),
				OpCode:     actions.CreateRollup,
			},
			&actions.RollupRegistration{
				Info: hactions.RollupInfo{
					Namespace:           []byte("nkit2"),
					FeeRecipient:        rsender,
					AuthoritySEQAddress: rsender,
				},
				Namespace: []byte("nkit2"),
				OpCode:    actions.DeleteRollup,
			},
		}

		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())

		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.True(result.Success)
		require.Empty(result.Error)
	})

	ginkgo.It("test tx fail for wrong authority modifications in rollup registration.", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		tpriv, err := bls.GeneratePrivateKey()
		require.NoError(err)

		pubKey := bls.PublicFromPrivateKey(tpriv)
		seqAddress := auth.NewBLSAddress(pubKey)

		currEpoch, err := instances[0].cli.GetCurrentEpoch()
		require.NoError(err)
		txActions := []chain.Action{
			&actions.RollupRegistration{
				Info: hactions.RollupInfo{
					Namespace:           []byte("nkit3"),
					FeeRecipient:        seqAddress,
					AuthoritySEQAddress: seqAddress,
				},
				StartEpoch: currEpoch + 5,
				Namespace:  []byte("nkit3"),
				OpCode:     actions.CreateRollup,
			},
			&actions.RollupRegistration{
				Info: hactions.RollupInfo{
					Namespace:           []byte("nkit3"),
					FeeRecipient:        seqAddress,
					AuthoritySEQAddress: rsender,
				},
				Namespace: []byte("nkit3"),
				OpCode:    actions.UpdateRollup,
			},
		}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())

		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.False(result.Success)
		require.Contains(string(result.Error), "not authorized")
	})

	ginkgo.It("test already existing namespace in rollup registration", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		tpriv, err := bls.GeneratePrivateKey()
		require.NoError(err)

		pubKey := bls.PublicFromPrivateKey(tpriv)
		seqAddress := auth.NewBLSAddress(pubKey)

		currEpoch, err := instances[0].cli.GetCurrentEpoch()
		require.NoError(err)
		txActions := []chain.Action{&actions.RollupRegistration{
			Info: hactions.RollupInfo{
				Namespace:           []byte("nkit"),
				FeeRecipient:        seqAddress,
				AuthoritySEQAddress: seqAddress,
			},
			StartEpoch: currEpoch + 5,
			Namespace:  []byte("nkit"),
			OpCode:     actions.CreateRollup,
		}}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())

		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.False(result.Success)
		require.Contains(string(result.Error), "namespace already registered")
	})

	var currEpoch uint64

	ginkgo.It("test create epoch exit for rollups", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		crEpoch, err := instances[0].cli.GetCurrentEpoch()
		currEpoch = crEpoch
		require.NoError(err)
		txActions := []chain.Action{
			&actions.RollupRegistration{
				Info: hactions.RollupInfo{
					Namespace:           []byte("nkit4"),
					FeeRecipient:        rsender,
					AuthoritySEQAddress: rsender,
				},
				Namespace:  []byte("nkit4"),
				StartEpoch: currEpoch + 10,
				OpCode:     actions.CreateRollup,
			},
			&actions.EpochExit{
				Info: storage.EpochInfo{
					Epoch:     currEpoch + 15,
					Namespace: []byte("nkit4"),
				},
				Epoch:  currEpoch + 15,
				OpCode: actions.CreateExit,
			},
		}

		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())
		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		fmt.Println("result: ", string(result.Error))
		require.True(result.Success)
		require.Empty(result.Error)
	})

	ginkgo.It("test delete epoch exit", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		txActions := []chain.Action{
			&actions.EpochExit{
				Info: storage.EpochInfo{
					Epoch:     currEpoch + 15,
					Namespace: []byte("nkit4"),
				},
				Epoch:  currEpoch + 15,
				OpCode: actions.DeleteExit,
			},
		}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())
		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		fmt.Println("result: ", string(result.Error))
		require.True(result.Success)
		require.Empty(result.Error)
	})

	ginkgo.It("test epoch exit only works for registered rollups", func() {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		txActions := []chain.Action{
			&actions.EpochExit{
				Info: storage.EpochInfo{
					Epoch:     currEpoch + 15,
					Namespace: []byte("nkit5"),
				},
				Epoch:  currEpoch + 15,
				OpCode: actions.CreateExit,
			},
		}
		parser, err := instances[0].tcli.Parser(ctx)
		require.NoError(err)
		_, tx, _, err := instances[0].cli.GenerateTransaction(ctx, parser, txActions, factory, 0)
		require.NoError(err)
		hutils.Outf("{{green}}txID of submitted data:{{/}}%s\n", tx.ID().String())
		err = instances[0].wsCli.RegisterTx(tx)
		require.NoError(err)
		txID, txErr, result, err := instances[0].wsCli.ListenTx(ctx)
		require.Equal(tx.ID(), txID)
		require.NoError(err)
		require.NoError(txErr)
		require.False(result.Success)
		require.Contains(string(result.Error), "namespace is not registered")
	})
	switch mode {
	case modeTest:
		hutils.Outf("{{yellow}}skipping bootstrap and state sync tests{{/}}\n")
		return
	}

	// Create blocks before bootstrapping starts
	count := 0
	ginkgo.It("supports issuance of 128 blocks", func() {
		count += generateBlocks(context.Background(), count, 128, instances, true)
	})

	// Ensure bootstrapping works
	var syncClient *rpc.JSONRPCClient
	var tsyncClient *trpc.JSONRPCClient
	ginkgo.It("can bootstrap a new node", func() {
		cluster, err := anrCli.AddNode(
			context.Background(),
			"bootstrap",
			execPath,
			trackSubnetsOpt,
			runner_sdk.WithChainConfigs(map[string]string{
				blockchainID: vmConfig,
			}),
		)
		require.NoError(err)
		awaitHealthy(anrCli)

		nodeURI := cluster.ClusterInfo.NodeInfos["bootstrap"].Uri
		uri := nodeURI + fmt.Sprintf("/ext/bc/%s", blockchainID)
		bid, err := ids.FromString(blockchainID)
		require.NoError(err)
		hutils.Outf("{{blue}}bootstrap node uri: %s{{/}}\n", uri)
		c := rpc.NewJSONRPCClient(uri)
		syncClient = c
		networkID, _, _, err := syncClient.Network(context.TODO())
		require.NoError(err)
		tc := trpc.NewJSONRPCClient(uri, networkID, bid)
		tsyncClient = tc
		instances = append(instances, instance{
			uri:  uri,
			cli:  c,
			tcli: tc,
		})
	})

	ginkgo.It("accepts transaction after it bootstraps", func() {
		acceptTransaction(syncClient, tsyncClient)
	})

	// Create blocks before state sync starts (state sync requires at least 256
	// blocks)
	//
	// We do 1024 so that there are a number of ranges of data to fetch.
	ginkgo.It("supports issuance of at least 1024 more blocks", func() {
		count += generateBlocks(context.Background(), count, 1024, instances, true)
		// TODO: verify all roots are equal
	})

	ginkgo.It("can state sync a new node when no new blocks are being produced", func() {
		cluster, err := anrCli.AddNode(
			context.Background(),
			"sync",
			execPath,
			trackSubnetsOpt,
			runner_sdk.WithChainConfigs(map[string]string{
				blockchainID: vmConfig,
			}),
		)
		require.NoError(err)

		awaitHealthy(anrCli)

		nodeURI := cluster.ClusterInfo.NodeInfos["sync"].Uri
		uri := nodeURI + fmt.Sprintf("/ext/bc/%s", blockchainID)
		bid, err := ids.FromString(blockchainID)
		require.NoError(err)
		hutils.Outf("{{blue}}sync node uri: %s{{/}}\n", uri)
		syncClient = rpc.NewJSONRPCClient(uri)
		networkID, _, _, err := syncClient.Network(context.TODO())
		require.NoError(err)
		tsyncClient = trpc.NewJSONRPCClient(uri, networkID, bid)
	})

	ginkgo.It("accepts transaction after state sync", func() {
		acceptTransaction(syncClient, tsyncClient)
	})

	ginkgo.It("can pause a node", func() {
		// shuts down the node and keeps all db/conf data for a proper restart
		_, err := anrCli.PauseNode(
			context.Background(),
			"sync",
		)
		require.NoError(err)

		awaitHealthy(anrCli)

		ok, err := syncClient.Ping(context.Background())
		require.Error(err)
		require.False(ok)
	})

	ginkgo.It("supports issuance of 256 more blocks", func() {
		count += generateBlocks(context.Background(), count, 256, instances, true)
		// TODO: verify all roots are equal
	})

	ginkgo.It("can re-sync the restarted node", func() {
		_, err := anrCli.ResumeNode(
			context.Background(),
			"sync",
		)
		require.NoError(err)

		awaitHealthy(anrCli)
	})

	ginkgo.It("accepts transaction after restarted node state sync", func() {
		acceptTransaction(syncClient, tsyncClient)
	})

	ginkgo.It("state sync while broadcasting transactions", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			// Recover failure if exits
			defer ginkgo.GinkgoRecover()

			count += generateBlocks(ctx, count, 0, instances, false)
		}()

		// Give time for transactions to start processing
		time.Sleep(5 * time.Second)

		// Start syncing node
		cluster, err := anrCli.AddNode(
			context.Background(),
			"sync_concurrent",
			execPath,
			trackSubnetsOpt,
			runner_sdk.WithChainConfigs(map[string]string{
				blockchainID: vmConfig,
			}),
		)
		require.NoError(err)
		awaitHealthy(anrCli)

		nodeURI := cluster.ClusterInfo.NodeInfos["sync_concurrent"].Uri
		uri := nodeURI + fmt.Sprintf("/ext/bc/%s", blockchainID)
		bid, err := ids.FromString(blockchainID)
		require.NoError(err)
		hutils.Outf("{{blue}}sync node uri: %s{{/}}\n", uri)
		syncClient = rpc.NewJSONRPCClient(uri)
		networkID, _, _, err := syncClient.Network(context.TODO())
		require.NoError(err)
		tsyncClient = trpc.NewJSONRPCClient(uri, networkID, bid)
		cancel()
	})

	ginkgo.It("accepts transaction after state sync concurrent", func() {
		acceptTransaction(syncClient, tsyncClient)
	})

	// TODO: restart all nodes (crisis simulation)
})

func awaitHealthy(cli runner_sdk.Client) {
	for {
		time.Sleep(healthPollInterval)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := cli.Health(ctx)
		cancel() // by default, health will wait to return until healthy
		if err == nil {
			return
		}
		hutils.Outf(
			"{{yellow}}waiting for health check to pass:{{/}} %v\n",
			err,
		)
	}
}

// generate blocks until either ctx is cancelled or the specified (!= 0) number of blocks is generated.
// if 0 blocks are specified, will just wait until ctx is cancelled.
func generateBlocks(
	ctx context.Context,
	cumulativeTxs int,
	blocksToGenerate uint64,
	instances []instance,
	failOnError bool,
) int {
	require := require.New(ginkgo.GinkgoT())

	_, lastHeight, _, err := instances[0].cli.Accepted(context.Background())
	require.NoError(err)
	parser, err := instances[0].tcli.Parser(context.Background())
	require.NoError(err)
	var targetHeight uint64
	if blocksToGenerate != 0 {
		targetHeight = lastHeight + blocksToGenerate
	}
	for ctx.Err() == nil {
		// Generate transaction
		other, err := ed25519.GeneratePrivateKey()
		require.NoError(err)
		submit, _, _, err := instances[cumulativeTxs%len(instances)].cli.GenerateTransaction(
			context.Background(),
			parser,
			[]chain.Action{&actions.Transfer{
				To:    auth.NewED25519Address(other.PublicKey()),
				Value: 1,
			}},
			factory,
			priorityFee,
		)
		if failOnError {
			require.NoError(err)
		} else if err != nil {
			hutils.Outf(
				"{{yellow}}unable to generate transaction:{{/}} %v\n",
				err,
			)
			time.Sleep(5 * time.Second)
			continue
		}

		// Broadcast transactions
		err = submit(context.Background())
		if failOnError {
			require.NoError(err)
		} else if err != nil {
			hutils.Outf(
				"{{yellow}}tx broadcast failed:{{/}} %v\n",
				err,
			)
			time.Sleep(5 * time.Second)
			continue
		}
		cumulativeTxs++
		_, height, _, err := instances[0].cli.Accepted(context.Background())
		if failOnError {
			require.NoError(err)
		} else if err != nil {
			hutils.Outf(
				"{{yellow}}height lookup failed:{{/}} %v\n",
				err,
			)
			time.Sleep(5 * time.Second)
			continue
		}
		if targetHeight != 0 && height > targetHeight {
			break
		} else if height > lastHeight {
			lastHeight = height
			hutils.Outf("{{yellow}}height=%d count=%d{{/}}\n", height, cumulativeTxs)
		}

		// Sleep for a very small amount of time to avoid overloading the
		// network with transactions (can generate very fast)
		time.Sleep(10 * time.Millisecond)
	}
	return cumulativeTxs
}

func acceptTransaction(cli *rpc.JSONRPCClient, tcli *trpc.JSONRPCClient) {
	require := require.New(ginkgo.GinkgoT())

	parser, err := tcli.Parser(context.Background())
	require.NoError(err)
	for {
		// Generate transaction
		other, err := ed25519.GeneratePrivateKey()
		require.NoError(err)
		unitPrices, err := cli.UnitPrices(context.Background(), false)
		require.NoError(err)
		submit, tx, maxFee, err := cli.GenerateTransaction(
			context.Background(),
			parser,
			[]chain.Action{&actions.Transfer{
				To:    auth.NewED25519Address(other.PublicKey()),
				Value: sendAmount,
			}},
			factory,
			priorityFee,
		)
		require.NoError(err)
		hutils.Outf("{{yellow}}generated transaction{{/}} prices: %+v maxFee: %d\n", unitPrices, maxFee)

		// Broadcast and wait for transaction
		require.NoError(submit(context.Background()))
		hutils.Outf("{{yellow}}submitted transaction{{/}}\n")
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		success, _, err := tcli.WaitForTransaction(ctx, tx.ID())
		cancel()
		if err != nil {
			hutils.Outf("{{red}}cannot find transaction: %v{{/}}\n", err)
			continue
		}
		require.True(success)
		hutils.Outf("{{yellow}}found transaction{{/}}\n")
		break
	}
}
