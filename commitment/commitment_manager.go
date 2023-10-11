package commitment

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/heap"
	"github.com/AnomalyFi/hypersdk/vm"
	"github.com/AnomalyFi/nodekit-seq/sequencer"
	"github.com/AnomalyFi/nodekit-seq/types"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	"go.uber.org/zap"

	ethcommon "github.com/ethereum/go-ethereum/common"

	ethbind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tidwall/btree"
)

const (
	maxWarpResponse   = bls.PublicKeyLen + bls.SignatureLen
	minGatherInterval = 30 * 60 // 30 minutes
	initialBackoff    = 0
	backoffIncrease   = 5
	maxRetries        = 10
	maxOutstanding    = 8 // TODO: make a config
)

// CommitmentManager takes the new block commitments and submits them to L1
type CommitmentManager struct {
	vm *vm.VM
	//appSender common.AppSender

	l         sync.Mutex
	requestID uint32

	headers *types.ShardedMap[string, *chain.StatelessBlock] // Map block ID to block

	idsByHeight btree.Map[uint64, ids.ID] // Map block ID to block height

	//TODO at some point maybe change this to be a FIFOMAP for optimization
	pendingJobs *heap.Heap[*blockJob, int64]
	jobs        map[uint32]*blockJob

	done chan struct{}
}

type blockJob struct {
	blockId string
	retry   int
}

func NewCommitmentManager(vm *vm.VM) *CommitmentManager {
	headers := types.NewShardedMap[string, *chain.StatelessBlock](10000, 10, types.HashString)

	return &CommitmentManager{
		vm:          vm,
		pendingJobs: heap.New[*blockJob, int64](64, true),
		idsByHeight: btree.Map[uint64, ids.ID]{},

		headers: headers,
		jobs:    map[uint32]*blockJob{},
		done:    make(chan struct{}),
	}
}

func (w *CommitmentManager) Run() {
	//w.appSender = appSender

	w.vm.Logger().Info("starting commitment manager")
	defer close(w.done)

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			w.l.Lock()
			now := time.Now().Unix()
			for w.pendingJobs.Len() > 0 && len(w.jobs) < maxOutstanding {
				first := w.pendingJobs.First()
				if first.Val > now {
					break
				}
				w.pendingJobs.Pop()

				// Send request
				job := first.Item
				if err := w.request(context.Background(), job); err != nil {
					w.vm.Logger().Error(
						"unable to request signature",
						zap.Error(err),
					)
				}
			}
			l := w.pendingJobs.Len()
			w.l.Unlock()
			w.vm.Logger().Debug("checked for ready jobs", zap.Int("pending", l))
		case <-w.vm.StopChan():
			w.vm.Logger().Info("stopping commitment manager")
			return
		}
	}
}

func (w *CommitmentManager) AcceptBlock(blk *chain.StatelessBlock) error {
	id := blk.ID()
	w.headers.Put(id.String(), blk)
	w.idsByHeight.Set(blk.Hght, id)

	//TODO eventually figure out a way to not submit the empty blocks to save costs

	w.l.Lock()
	if w.pendingJobs.Has(id) {
		// We may already have enqueued a job when the block was accepted.
		//TODO do I need this or not?
		w.l.Unlock()
		return nil
	}
	w.pendingJobs.Push(&heap.Entry[*blockJob, int64]{
		ID: id,
		Item: &blockJob{
			id.String(),
			0,
		},
		Val:   time.Now().Unix() + initialBackoff,
		Index: w.pendingJobs.Len(),
	})
	w.l.Unlock()
	w.vm.Logger().Debug(
		"enqueued push job",
		zap.Stringer("blockId", id),
	)
	return nil
}

// you must hold [w.l] when calling this function
func (w *CommitmentManager) request(
	ctx context.Context,
	j *blockJob,
) error {
	//TODO do I need this requestID stuff or not?
	// requestID := w.requestID
	// w.requestID++
	// w.jobs[requestID] = j

	blk, success := w.headers.Get(j.blockId)

	if !success {
		return errors.New("invalid request to get block")
	}

	conn, err := ethclient.Dial("http://0.0.0.0:8545")

	priv, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	if err != nil {
		log.Fatalf("Failed to convert from hex to ECDSA: %v", err)
		return err
	}

	auth, err := ethbind.NewKeyedTransactorWithChainID(priv, big.NewInt(900))
	if err != nil {
		log.Fatalf("Failed to create authorized transactor: %v", err)
		return err
	}

	auth.GasLimit = 1_000_000

	//TODO change the address
	sequencerContractTest, err := sequencer.NewSequencer(ethcommon.HexToAddress("0x3a92c8145cb9694e2E52654707f3Fa71021fc4AC"), conn)

	// function call on `instance`. Retrieves pending name
	maxBlocks, err := sequencerContractTest.MAXBLOCKS(&ethbind.CallOpts{Pending: true})
	if err != nil {
		log.Fatalf("Failed to retrieve max blocks: %v", err)
		return err
	}
	fmt.Println("max blocks:", maxBlocks)

	contract_block_height, err := sequencerContractTest.BlockHeight(nil)
	if err != nil {
		log.Fatalf("Failed to retrieve max blocks: %v", err)
		return err
	}

	// The way SEQ works with producing empty blocks means that all we need to do is once a block has 1 or more transactions we check the height on the L1 contract
	// and if it is less so the L1 is behind SEQ then we can submit a batch

	blkHeight := big.NewInt(int64(blk.Hght))
	fmt.Println("Current Block Height:", contract_block_height)
	fmt.Printf("Update SEQ Height: %d\n", blk.Hght)
	if contract_block_height.Cmp(blkHeight) < 0 {
		// fmt.Println("Current Block Height:", contract_block_height)
		// fmt.Printf("Update SEQ Height: %x\n", blk.Hght)

		firstBlock := uint64(contract_block_height.Int64())

		blocks := make([]sequencer.SequencerWarpBlock, 0)

		w.idsByHeight.Ascend(firstBlock, func(heightKey uint64, id ids.ID) bool {
			//Does heightKey match the given block's height for the id
			blockTemp, success := w.headers.Get(id.String())
			if !success {
				return success
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

			if blockTemp.Hght > blk.Hght {
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
			return true

		})

		tx, err := sequencerContractTest.NewBlocks(auth, blocks)
		if err != nil {
			fmt.Println("An error happened:", err)
			return err
		}

		fmt.Printf("Update pending: 0x%x\n", tx.Hash())

	}
	return nil
}

func (w *CommitmentManager) Done() {
	<-w.done
}
