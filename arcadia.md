# Arcadia

Arcadia facilitates the Shared Block Production on The Composable Network for participating Rollups. Block Production on Arcadia is divided into 12s(6 SEQ blocks) Epochs with a single block builder building blocks for all the participating rollups in that Epoch. Winning block builder is selected through a Ahead Of Time Auction on Arcadia. Rollup blocks are built as chunks and preconfed by validators. All txs in preconfed chunks will be included in SEQ blocks in later time.

## Rollups:
- Rollups opt for shared block production by Arcadia ahead of time from a given epoch and added to list of participating rollups for the shared auction.
- Rollups get synchronous atomic composability for the blocks built during the participating epoch.
- Once opt in for shared block production, rollups continue to exist in the list of participating rollups for future epochs, until they entirely opt out or opt out for a specific epoch.
- Rollups opt in and out entirely through `RegisterRollup` action tx.
- Rollups opt out for an epoch through `EpochExit` action tx.

## Ahead Of Time Auction:
- Auction for rights to build blocks for participating rollups in Epoch K starts one Epoch earlier and ends with the proudction of 4th block of SEQ in Epoch K-1.
- Rollups can Opt In or Out for Epoch K, before the end of Epoch K-2. The list of participating rollups for Epoch K finalises with the end of Epoch K-2 and builders bid for rights to build blocks for all participating rollups, based on the perceived value accural for Epoch K and private order flow deals.
- With 4th block of SEQ in Epoch k-1 produced, winning bid is sent to be included in next SEQ block as `Auction` action tx, to deduct bid amount from the builder.
- All participating entities are notified about the bid winner, once the `Auction` action tx makes through a SEQ block.

## Chunks:
- In Arcadia txs get preconfed as chunks, rather than blocks. 
- There are two defined chunk types: i. Top Of Block(ToB) ii. Rest Of Block(RoB). A chunk consists a set of txs or tx bundles, depending on it being a ToB or RoB.
- SEQ supports a single rollup tx or cross chain bundle in a single tx as multi-action tx is supported.
- A ToB contains cross rollup bundles only. But necessarily a ToB need not contain txs for all participating rollups, but can contain any subset of the list. A RoB contains txs for a single rollup.
- Chunks get preconfed by validators and made available for rollups to pull.
- A Rollup block can have multiple ToB chunks, but only one RoB chunk. 

## Block Builders:
- Block builders participate in auction after confirming their ability to build blocks for participating rollups.
- They build ToB chunks for different subsets of participating rollups and RoB chunks for each rollup, and send them with rollup block numbers and chunk nonce to Arcadia continuosly to get them preconfed by SEQ validators.
- Every rollup block, ideally contains many ToBs and one RoB. ToBs are ordered by the chunk nonce included along with the chunk followed by RoB.
  
ToB chunks containig different subset of rollups:
<p align="center">
    <img src="./assets/tob.png" width="40%">
</p>

A rollup block built by builder:
<p align="center">
  <img src="./assets/rollup_block.png" width="50%">
</p>
## SEQ Validators:
- SEQ validators register with Arcadia to receive chunks for preconfing.
- SEQ validators check few assertions on chunks, and issue a chunk cert if the assertions are satisfied. 
- Valdiators store the signature verified txs in a emap, to prevent duplicate txs and ease signature verification while accepting the block.
- SEQ blocks are produced every 2 seconds, validators fetch `SequencerMsg` txs payload from Arcadia to get included in the SEQ block and fill the rest of the block with `non SequencerMsg` txs.

## E2E Flow:
![E2E flow](./assets/e2e.png)
