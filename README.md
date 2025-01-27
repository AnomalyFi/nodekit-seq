## Status
`SeqVM` is considered **ALPHA** software and is not safe to use in
production. The framework is under active development and may change
significantly over the coming months as its modules are optimized and
audited.

## Features
### Shared Sequencer
The `SeqVM` is built from the ground up with the shared sequencer built directly into the chain 
enabling decentralization from the start. This enables users to easily send messages from rollups like NodeKit chain to the shared sequencer. The contents of `SequencerMSG` is just `ChainId|Data|FromAddress` where the data is the transaction data 
from the rollup translated into a byte[]. 
### Arcadia
Rollups using Arcadia, will be able to produce blocks with sub second block times without losing synchronous atomic composability with other rollups using Arcadia, all thanks to Arcadia's ability to preconf rollup block chunks in 250ms and make the block available for rollups to pull instantly after preconfing.

> Read more about arcadia [here](./arcadia.md).

## Demos
The first step to running these demos is to launch your own `SeqVM` Subnet. You
can do so by running the following command from this location (may take a few
minutes):
```bash
./scripts/run.sh;
```

_By default, this allocates all funds on the network to
`seq1qrzvk4zlwj9zsacqgtufx7zvapd3quufqpxk5rsdd4633m4wz2fdjlydh3t`. The private
key for this address is
`0x323b1d8f4eed5f0da9da93071b034f2dce9d2d22692c172f3cb252a64ddfafd01b057de320297c29ad0c1f589ea216869cf1938d88c9fbd70d6748323dbf2fa7`.
For convenience, this key has is also stored at `demo.pk`._

_If you don't need 2 Subnets for your testing, you can run `MODE="run-single"
./scripts/run.sh`._

To make it easy to interact with the `SeqVM`, we implemented the `seq-cli`.
Next, you'll need to build this. You can use the following command from this location
to do so:
```bash
./scripts/build.sh
```

_This command will put the compiled CLI in `./build/seq-cli`._

Lastly, you'll need to add the chains you created and the default key to the
`seq-cli`. You can use the following commands from this location to do so:
```bash
./build/seq-cli key import demo.pk
./build/seq-cli chain import-anr
```

_`chain import-anr` connects to the Avalanche Network Runner server running in
the background and pulls the URIs of all nodes tracking each chain you
created._

## View Your Balance
Now, let's check that the mint worked right by checking our balance. You can do
so by running the following command from this location:
```bash
./build/seq-cli key balance
```

When you are done, the output should look something like this:
```
database: .seq-cli
address: seq1qrzvk4zlwj9zsacqgtufx7zvapd3quufqpxk5rsdd4633m4wz2fdjlydh3t
chainID: Em2pZtHr7rDCzii43an2bBi1M2mTFyLN33QP1Xfjy7BcWtaH9
assetID (use SEQ for native seq): 27grFs9vE2YP9kwLM5hQJGLDvqEY9ii71zzdoRHNGC4Appavug
balance: 10000 27grFs9vE2YP9kwLM5hQJGLDvqEY9ii71zzdoRHNGC4Appavug
```

## Bonus: Watch Activity in Real-Time
To provide a better sense of what is actually happening on-chain, the
`seq-cli` comes bundled with a simple explorer that logs all blocks/txs that
occur on-chain. You can run this utility by running the following command from
this location:
```bash
./build/seq-cli chain watch
```

If you run it correctly, you'll see the following input (will run until the
network shuts down or you exit):
```
database: .seq-cli
available chains: 2 excluded: []
0) chainID: Em2pZtHr7rDCzii43an2bBi1M2mTFyLN33QP1Xfjy7BcWtaH9
1) chainID: cKVefMmNPSKmLoshR15Fzxmx52Y5yUSPqWiJsNFUg1WgNQVMX
select chainID: 0
watching for new blocks on Em2pZtHr7rDCzii43an2bBi1M2mTFyLN33QP1Xfjy7BcWtaH9 👀
height:13 txs:1 units:488 root:2po1n8rqdpNuwpMGndqC2hjt6Xa3cUDsjEpm7D6u9kJRFEPmdL avg TPS:0.026082
✅ KwHcsy3TXcnDyoMNmpMYC4EUvXPDPCoTK8YaEigG7nWwwdi1n actor: seq1rvzhmceq997zntgvravfagsks6w0ryud3rylh4cdvayry0dl97nsjzf3yp units: 496 summary (*actions.SequencerMsg): [data: ]

```

### Running a Load Test
_Before running this demo, make sure to stop the network you started using
`killall avalanche-network-runner`._

The `SeqVM` load test will provision 5 `SeqVMs` and process 500k transfers
on each between 10k different accounts.

```bash
./scripts/tests.load.sh
```

_This test SOLELY tests the speed of the `SeqVM`. It does not include any
network delay or consensus overhead. It just tests the underlying performance
of the `hypersdk` and the storage engine used (in this case MerkleDB on top of
Pebble)._

#### Measuring Disk Speed
This test is extremely sensitive to disk performance. When reporting any TPS
results, please include the output of:

```bash
./scripts/tests.disk.sh
```

_Run this test RARELY. It writes/reads many GBs from your disk and can fry an
SSD if you run it too often. We run this in CI to standardize the result of all
load tests._

## Zipkin Tracing
To trace the performance of `SeqVM` during load testing, we use `OpenTelemetry + Zipkin`.

To get started, startup the `Zipkin` backend and `ElasticSearch` database (inside `hypersdk/trace`):
```bash
docker-compose -f trace/zipkin.yml up
```
Once `Zipkin` is running, you can visit it at `http://localhost:9411`.

Next, startup the load tester (it will automatically send traces to `Zipkin`):
```bash
TRACE=true ./scripts/tests.load.sh
```

When you are done, you can tear everything down by running the following
command:
```bash
docker-compose -f trace/zipkin.yml down
```

## Deploying to a Devnet
_In the world of Avalanche, we refer to short-lived, test Subnets as Devnets._

To programmatically deploy `SeqVM` to a distributed cluster of nodes running on
your own custom network or on Fuji, check out this [doc](DEVNETS.md).

## Future Work
_If you want to take the lead on any of these items, please
[start a discussion](https://github.com/AnomalyFi/nodekit-seq/discussions) or reach
out on the NodeKit Discord._
