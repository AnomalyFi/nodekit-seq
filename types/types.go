package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	// "github.com/AnomalyFi/hypersdk/chain"
	"github.com/ava-labs/avalanchego/ids"
	// "github.com/chainbound/shardmap"
)

// ! TODO I should use one or two shardmap across the jsonrpc_server and the commitmentmanager to simplify it
// TODO I can always optimize this later but it just needs to work for now
// type BlockDTO struct {
// 	//The strings here are just ids.ID converted to strings to maintain compatiblity with this library
// 	headers shardmap.ShardedMap[string, *chain.StatefulBlock] // Map block ID to block header

// 	blocksWithValidTxs shardmap.ShardedMap[string, *SequencerBlock] // Map block ID to block header

// }

type SEQTransaction struct {
	Namespace   string `json:"namespace"`
	Tx_id       string `json:"tx_id"`
	Index       uint64 `json:"tx_index"`
	Transaction []byte `json:"transaction"`
}

type SequencerBlock struct {
	StateRoot ids.ID                       `json:"state_root"`
	Prnt      ids.ID                       `json:"parent"`
	Tmstmp    int64                        `json:"timestamp"`
	Hght      uint64                       `json:"height"`
	Txs       map[string][]*SEQTransaction `json:"transactions"`
}

// A BigInt type which serializes to JSON a a hex string.
type U256 struct {
	big.Int
}

func NewU256() *U256 {
	return new(U256)
}

func (i *U256) SetBigInt(n *big.Int) *U256 {
	i.Int.Set(n)
	return i
}

func (i *U256) SetUint64(n uint64) *U256 {
	i.Int.SetUint64(n)
	return i
}

func (i *U256) SetBytes(buf [32]byte) *U256 {
	i.Int.SetBytes(buf[:])
	return i
}

func (i U256) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("0x%s", i.Text(16)))
}

func (i *U256) UnmarshalJSON(in []byte) error {
	var s string
	if err := json.Unmarshal(in, &s); err != nil {
		return err
	}
	if _, err := fmt.Sscanf(s, "0x%x", &i.Int); err != nil {
		return err
	}
	return nil
}
