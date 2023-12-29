package commitment

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/vm"
	"github.com/AnomalyFi/nodekit-seq/sequencer"
	"github.com/AnomalyFi/nodekit-seq/types"

	"github.com/ava-labs/avalanchego/ids"

	ethbind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tidwall/btree"
)

// CommitmentManager takes the new block commitments and submits them to L1
type CommitmentManager struct {
	vm *vm.VM

	headers *types.ShardedMap[string, *chain.StatelessBlock] // Map block ID to block

	idsByHeight btree.Map[uint64, ids.ID] // Map block ID to block height

	// TODO at some point maybe change this to be a FIFOMAP for optimization
	done chan struct{}
}

func NewCommitmentManager(vm *vm.VM) *CommitmentManager {
	headers := types.NewShardedMap[string, *chain.StatelessBlock](10000, 10, types.HashString)

	return &CommitmentManager{
		vm:          vm,
		idsByHeight: btree.Map[uint64, ids.ID]{},
		headers:     headers,
		done:        make(chan struct{}),
	}
}

func (w *CommitmentManager) Run() {
	w.vm.Logger().Info("starting commitment manager")

	conn, err := ethclient.Dial("https://devnet.nodekit.xyz")

	sequencerContractTest, err := sequencer.NewSequencer(ethcommon.HexToAddress("0x5FbDB2315678afecb367f032d93F642f64180aa3"), conn)

	// function call on `instance`. Retrieves pending name
	maxBlocks, err := sequencerContractTest.MAXBLOCKS(&ethbind.CallOpts{Pending: true})
	if err != nil {
		log.Fatalf("Failed to retrieve max blocks: %v", err)
	}
	fmt.Println("max blocks:", maxBlocks)

	if !(w.idsByHeight.Len() > 1) {
		time.Sleep(2 * time.Second)
	}

	w.Commit(maxBlocks, sequencerContractTest, conn)

}

func (w *CommitmentManager) Commit(maxBlocks *big.Int, seq *sequencer.Sequencer, client *ethclient.Client) {
	for {
		if err := w.SyncRequest(maxBlocks, seq, client); err != nil {
			fmt.Printf("Failed to Sync %v\n", err)

			// Wait to avoid spam
			time.Sleep(1 * time.Second)
		}
	}

}

// func (w *CommitmentManager) AcceptBlock(blk *chain.StatelessBlock) error {
// 	id := blk.ID()
// 	w.headers.Put(id.String(), blk)
// 	w.idsByHeight.Set(blk.Hght, id)

// 	// w.vm.Logger().Debug(
// 	// 	"Added new block",
// 	// 	zap.Stringer("blockId", id),
// 	// )
// 	return nil
// }

// you must hold [w.l] when calling this function
func (w *CommitmentManager) SyncRequest(
	maxBlocks *big.Int,
	seq *sequencer.Sequencer,
	client *ethclient.Client,
) error {

	//TODO may need to change
	priv, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		log.Fatalf("Failed to convert from hex to ECDSA: %v", err)
		return err
	}

	//! TODO just changed the chainId so this should fix it
	auth, err := ethbind.NewKeyedTransactorWithChainID(priv, big.NewInt(32382))
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
		return err
	}

	//TODO! Just changed this because it was potentially causing issues on OP Stack
	//auth.GasLimit = 1_000_000
	auth.GasLimit = 300_000

	contract_block_height, err := seq.BlockHeight(nil)
	if err != nil {
		log.Fatalf("Failed to retrieve max blocks: %v", err)
		return err
	}

	// The way SEQ works with producing empty blocks means that all we need to do is once a block has 1 or more transactions we check the height on the L1 contract
	// and if it is less so the L1 is behind SEQ then we can submit a batch

	var blkHeight uint64

	//TODO fix this later

	//pivot := uint64(1000000000)

	pivot, _, _ := w.idsByHeight.Max()

	w.idsByHeight.Descend(pivot, func(height uint64, blk ids.ID) bool {
		blkHeight = height
		return false

	})

	blkHeightBig := big.NewInt(int64(blkHeight))
	fmt.Println("Current Block Height:", contract_block_height)
	fmt.Printf("Update SEQ Height: %d\n", blkHeight)
	if contract_block_height.Cmp(blkHeightBig) <= 0 {
		// fmt.Println("Current Block Height:", contract_block_height)
		// fmt.Printf("Update SEQ Height: %x\n", blk.Hght)

		firstBlock := uint64(contract_block_height.Int64())

		blocks := make([]sequencer.SequencerWarpBlock, 0)
		//TODO need to make this only do maximum of size 500

		w.idsByHeight.Ascend(firstBlock, func(heightKey uint64, id ids.ID) bool {
			// Does heightKey match the given block's height for the id
			if len(blocks) >= 500 {
				return false
			}

			blockTemp, success := w.headers.Get(id.String())
			if !success {
				return success
			}

			//TODO swapped these 2 functions so now it exits earlier. Need to test
			if blockTemp.Hght >= blkHeight {
				root := types.NewU256().SetBytes(blockTemp.StateRoot)
				bigRoot := root.Int
				parentRoot := types.NewU256().SetBytes(blockTemp.Prnt)
				bigParentRoot := parentRoot.Int

				blocks = append(blocks, sequencer.SequencerWarpBlock{
					Height:     big.NewInt(int64(blockTemp.Hght)),
					BlockRoot:  &bigRoot,
					ParentRoot: &bigParentRoot,
				})
				return false
			}

			if blockTemp.Hght == heightKey {
				root := types.NewU256().SetBytes(blockTemp.StateRoot)
				bigRoot := root.Int
				parentRoot := types.NewU256().SetBytes(blockTemp.Prnt)
				bigParentRoot := parentRoot.Int

				blocks = append(blocks, sequencer.SequencerWarpBlock{
					Height:     big.NewInt(int64(blockTemp.Hght)),
					BlockRoot:  &bigRoot,
					ParentRoot: &bigParentRoot,
				})
			}

			return true
		})

		fmt.Println("Created Block Batch")

		tx, err := seq.NewBlocks(auth, blocks)
		if err != nil {
			fmt.Println("An error happened:", err)
			return err
		}

		//ch := make(chan *ethtypes.Transaction)
		for i := 1; i < 10; i++ {
			tx, pending, _ := client.TransactionByHash(context.TODO(), tx.Hash())
			if !pending {
				fmt.Printf("Update Sucessful: 0x%x\n", tx.Hash())

			}

			time.Sleep(time.Second * 1)
		}

	}
	return nil
}

// func (w *CommitmentManager) Done() {
// 	<-w.done
// }
