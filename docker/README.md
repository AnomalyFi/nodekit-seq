# Local Nodekit Network with Docker

This guide explains how to set up a local Avalanche test network running a SEQ blockchain + Ethereum node with Docker Compose and the `ash.avalanche` Ansible collection.

## Requirements

- Python >=3.9 with `venv` module installed
- [Docker](https://docs.docker.com) installed (see [Get Docker](https://docs.docker.com/get-docker/))
- [Packer](https://developer.hashicorp.com/packer) installed (see [Install Packer](https://developer.hashicorp.com/packer/downloads))
- The Avalanche VM to install has to be specified by enriching the `avalanchego_vms_list` variable in [./inventory/local/avalanche_nodes.yml](./inventory/local/group_vars/avalanche_nodes.yml). Example:

  ```yaml
  avalanchego_vms_list:
    tokenvm:
      # download_url and path are mutually exclusive
      download_url: https://github.com/AshAvalanche/hypersdk/releases/download
      # path: "{{ inventory_dir }}/../../files" # tokenvm_0.0.999_linux_amd64.tar.gz
      id: tHBYNu8ikqo4MWMHehC9iKB9mR5tB3DWzbkYmTfe9buWQ5GZ8
      # Used in Ash CLI
      ash_vm_type: Custom
      binary_filename: tokenvm
      aliases:
        - tokenvm
      versions_comp:
        0.0.999:
          ge: 1.10.10
          le: 1.10.10
  ```

Where:
- `download_url` is a URL to download the VM archive from a GitHub release
- **OR** `path` is a path to a local directory containing the VM archive (e.g.: `"{{ inventory_dir }}/../../files/tokenvm_0.0.999_linux_amd64.tar.gz"`)
- `id` is the VM ID
- `ash_vm_type` is the VM type used in the Ash CLI, should always be `Custom` if not using `SubnetEVM`
- `aliases` is a list of aliases for the VM
- `versions_comp` is a dictionary of version constraints for the VM. The `ge` and `le` keys are the constraints for the AvalancheGo version

**Note:** In the example above, the `tokenvm` VM release [`999`](https://github.com/AshAvalanche/hypersdk/releases/tag/v0.0.999) is actually a **Nodekit SEQ** release based on branch [`ash`](https://github.com/AnomalyFi/nodekit-seq/tree/ash) (see [`../scripts/build.ash_compatible.release.sh`](../scripts/build.ash_compatible.release.sh) to build an Ash-compatible release).

## Setup the environment

1. Setup the Python venv

    ```bash
    cd docker
    ./setup.sh
    ```

2. Activate Python venv:

   ```bash
   source .venv/bin/activate
   ```

3. Install the `ash.avalanche` collection:

   ```bash
   ansible-galaxy collection install git+https://github.com/AshAvalanche/ansible-avalanche-collection.git,v0.12.3
   ```


## Build the Docker images

The `avalanche-node-*.pkr.hcl` Packer templates allow to build Docker images for the 5 Avalanche nodes of the `local` network with:

- 1 fixed version of AvalancheGo
- 1 fixed version of an Avalanche VM

Out of simplicity, the `build_images.sh` script is provided:

```bash
# Set the AvalancheGo version
export AVALANCHEGO_VERSION=1.10.10
# Set the Avalanche VM to install with its version
export AVALANCHEGO_VM_NAME=tokenvm
export AVALANCHEGO_VM_VERSION=0.0.999

./build_images.sh
```

**Note:** The environment variables' default values are defined in `build.sh` and can be overridden by setting them before running the script.

The build will publish the following Docker images to the local Docker registry:

```bash
$ docker image ls
REPOSITORY                              TAG                        IMAGE ID       CREATED              SIZE
ash/avalanche-node-local-validator05    1.10.10-tokenvm-0.0.999    4f115dd01d73   36 seconds ago       520MB
ash/avalanche-node-local-validator04    1.10.10-tokenvm-0.0.999    6aaabf544256   48 seconds ago       520MB
ash/avalanche-node-local-validator03    1.10.10-tokenvm-0.0.999    33d3e9831bd7   59 seconds ago       520MB
ash/avalanche-node-local-validator02    1.10.10-tokenvm-0.0.999    e7ae745b1625   About a minute ago   520MB
ash/avalanche-node-local-validator01    1.10.10-tokenvm-0.0.999    8a67defee903   About a minute ago   520MB
```

## Start the local Avalanche test network with Docker Compose

To start the local Avalanche test network with Docker Compose, simply run:

```bash
docker compose up -d
```

The [`compose.yml`](./compose.yml) contains:
- 5 Avalanche nodes services `avalanche-local-validator0[1-5]`
- 1 Ethereum node service `l1`

**Note:** The Dockerfile for the `l1` container can be found at [`./eth-l1/Dockerfile-l1`](./eth-l1/Dockerfile-l1). 

## Wait for the network to be ready

```bash
# Wait for the network to be ready (i.e. all Avalanche nodes are bootstrapped and Rewarding stake: 100%, see output below)
docker exec -t ash-avalanche-local-validator01 ash avalanche node info
```

Output should be:

```
Node '127.0.0.1:9650':
  ID:            NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg
  Signer (BLS):
    Public key:  0xae13870387588ffa555fd63226b5cd7634fb6b9e5c4162a5eccc20740566790189d5520e2c8e3bd535c9922639c4d86e
    PoP:         0x99dcf81f0aab0d6cfa9a3365fa02d38116f7f00c0f00563942fdff3716d1c0bb39bef60e04459a5094eddee372ff3ce71328a58548c7aaa022b7571dc66540dc50eacb2a36b265cad674d5b0d6ad354c45802ed0adf998983fc757b8a52d2be8
  Network:       local
  Public IP:     177.17.0.11
  Staking port:  9651
  Versions:
    AvalancheGo:  avalanche/1.10.10
    Database:     v1.4.5
    RPC Protocol: 28
    Git commit:   73ae39bded4f1c8bf8aa58c20f93ce7456a612e9
    VMs:
      AvalancheVM: v1.10.10
      Coreth:      v0.12.5
      PlatformVM:  v1.10.10
      Subnet VMs:
        'tHBYNu8ikqo4MWMHehC9iKB9mR5tB3DWzbkYmTfe9buWQ5GZ8': v0.0.1
  Uptime:
    Rewarding stake:  100%
    Weighted average: 100%
```

## Creating a local Subnet

To create a local Subnet, you can use the same playbook as for the Multipass-based local test network:

```bash
ansible-playbook ash.avalanche.create_subnet -i inventory/local
```

Take note of the Subnet ID and Blockchain ID in the output:

```bash
TASK [Display Subnet information]
ok: [validator01] =>
  msg:
    blockchains:
    - id: 2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au                    <=== Blockchain ID
      name: seqchain
      subnetID: 29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL
      vmID: tHBYNu8ikqo4MWMHehC9iKB9mR5tB3DWzbkYmTfe9buWQ5GZ8
      vmType: SubnetEVM
    controlKeys:
    - P-local18jma8ppw3nhx5r4ap8clazz0dps7rv5u00z96u
    id: 29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL                      <=== Subnet ID
    pendingValidators:
    - connected: false
      endTime: 1709056176
      nodeID: NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg
      stakeAmount: 100
      startTime: 1708451376
      txID: 2j8kj87kyWayyaXX9WNaYxC8JujjfqV1DbKqnTRiLAqNKFSLkJ
      weight: 100
    subnetType: Permissioned
    threshold: 1
    validators: []
```

**Note:** To customize the Subnet genesis, you can edit the `subnet_blockchains_list` variable in the [`./inventory/local/group_vars/subnet_txs_host.yml`](./inventory/local/group_vars/subnet_txs_host.yml) file.

## Reconfigure the Avalanche nodes to track the new Subnet

To track the newly created Subnet, we need to add the Subnet ID to the `track-subnets` configuration parameter of the Avalanche nodes.

1. Adding the Subnet ID to the `track-subnets` configuration parameter of the Avalanche nodes:

   ```bash
   sed -i 's/"track-subnets": ".*"/"track-subnets": "29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL"/' "conf/bootstrap/conf/node.json";
   sed -i 's/"track-subnets": ".*"/"track-subnets": "29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL"/' "conf/node/conf/node.json";
   ```

2. Renaming the chains config directory if needed (has to be named after the chain ID):
   ```bash
   mv conf/bootstrap/conf/chains/2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au conf/bootstrap/conf/chains/${CHAIN_ID}
   mv conf/node/conf/chains/2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au conf/node/conf/chains/${CHAIN_ID}
   ```

3. Renaming the subnet config file if needed (has to be named after the Subnet ID):
   ```bash
   mv conf/bootstrap/conf/subnets/29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL.json conf/bootstrap/conf/subnets/${SUBNET_ID}.json
   mv conf/node/conf/subnets/29uVeLPJB1eQJkzRemU8g8wZDw5uJRqpab5U2mX9euieVwiEbL.json conf/node/conf/subnets/${SUBNET_ID}.json
   ```

4. Restarting the Avalanche nodes:

   ```bash
   docker compose restart avalanche-local-validator01 avalanche-local-validator02 avalanche-local-validator03 avalanche-local-validator04 avalanche-local-validator05
   ```

**Note:** Steps 2. and 3. are only needed if the chain ID and subnet ID are not the default ones.

## Chain import

```bash
../build/token-cli chain import
chainID: 2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au
✔ uri: http://177.17.0.11:9650/ext/bc/2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au
```

**Note:** `177.17.0.11` is the IP address of the `avalanche-local-validator01` container.

## Chain watch

```bash
../build/token-cli chain watch
database: .token-cli
available chains: 1 excluded: []
1) chainID: 2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au
select chainID: 0 [auto-selected]
uri: http://177.17.0.11:9650/ext/bc/2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au
Here is network Id: %d 12345
Here is uri: %s http://177.17.0.11:9650/ext/bc/2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au
watching for new blocks on 2ArqB8j5FWQY9ZBtA3QFJgiH9EmXzbqGup5kuyPQZVZcL913Au 👀
height:54 l1head:%!s(int64=106) txs:0 root:kMYb8yTR9pbtfFs5hfuEZLDjVjWnepSjpRE3kkXrEeJ2JyPAA blockId:KSLxXyN7XT67n5ubbzHzW7afA6tTjn5hmyqxwr5o4JRSt5wGi size:0.10KB units consumed: [bandwidth=0 compute=0 storage(read)=0 storage(create)=0 storage(modify)=0] unit prices: [bandwidth=100 compute=100 storage(read)=100 storage(create)=100 storage(modify)=100]
height:55 l1head:%!s(int64=107) txs:0 root:23ArBGmR9DQQLab1icAGFZbKd941FdrZuEy5oY2Yj6ZtKDri9M blockId:22zLgkd5bDmgu2qHqDBYQognAktpnp946vHx1MxDGYG38WCZPe size:0.10KB units consumed: [bandwidth=0 compute=0 storage(read)=0 storage(create)=0 storage(modify)=0] unit prices: [bandwidth=100 compute=100 storage(read)=100 storage(create)=100 storage(modify)=100] [TPS:0.00 latency:110ms gap:2474ms]
height:56 l1head:%!s(int64=108) txs:0 root:JZ5w66ZU2MbyaVGqEKoLZ92XHvRnht9Wy79mxNRXJ4k9ZNU3z blockId:K6f61bRPavvJDxuKD4rBmBDLoNTptJWJvJD9zxFcevL3Scjqj size:0.10KB units consumed: [bandwidth=0 compute=0 storage(read)=0 storage(create)=0 storage(modify)=0] unit prices: [bandwidth=100 compute=100 storage(read)=100 storage(create)=100 storage(modify)=100] [TPS:0.00 latency:125ms gap:2533ms]
height:57 l1head:%!s(int64=109) txs:0 root:2EmEPrLQL2a8n8MNPsFExLejWUaZsPsW9tZsnhGJSp4Btmh65q blockId:wNFtM2EnQy1u19sPZzo9cfgfesaSWJPgESzjU6gSU1JZcb328 size:0.10KB units consumed: [bandwidth=0 compute=0 storage(read)=0 storage(create)=0 storage(modify)=0] unit prices: [bandwidth=100 compute=100 storage(read)=100 storage(create)=100 storage(modify)=100] [TPS:0.00 latency:109ms gap:2486ms]
height:58 l1head:%!s(int64=109) txs:0 root:2DdjYeckENJmq92HWsSs8D6DFQpqiCtQFjDMaMWXrjFfQPbwQw blockId:2iJPUrPDSbNjUXQd8tfkQFGyJ7nyxjtorha2TLJdx2UWrrX6HT size:0.10KB units consumed: [bandwidth=0 compute=0 storage(read)=0 storage(create)=0 storage(modify)=0] unit prices: [bandwidth=100 compute=100 storage(read)=100 storage(create)=100 storage(modify)=100] [TPS:0.00 latency:118ms gap:2532ms]
height:59 l1head:%!s(int64=110) txs:0 root:2YmfCHEBQ5GrDqsRydPQcMTacYS1QaCdaVsPw5JkTrof6H6Jjf blockId:NppsG239uko64QQHWpPEDx9GTuQvwVRTN5wvFHmjZPgpRDBhh size:0.10KB units consumed: [bandwidth=0 compute=0 storage(read)=0 storage(create)=0 storage(modify)=0] unit prices: [bandwidth=100 compute=100 storage(read)=100 storage(create)=100 storage(modify)=100] [TPS:0.00 latency:112ms gap:2500ms]
```

## Teardown

To stop and remove the local Avalanche test network and Ethereum node with Docker Compose, run:

```bash
docker compose down
```

**Note:** The Ethereum node data is persisted in the `eth-l1/l1_data` directory. To remove it, run:

```bash
sudo rm -rf eth-l1/l1_data
```
