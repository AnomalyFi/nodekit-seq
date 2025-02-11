package types

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/ava-labs/avalanchego/ids"

	hactions "github.com/AnomalyFi/hypersdk/actions"
)

type SEQTransaction struct {
	Namespace   string `json:"namespace"`
	TxID        string `json:"tx_id"`
	Index       uint64 `json:"tx_index"`
	Transaction []byte `json:"transaction"`
}

type SEQTransactionResponse struct {
	Txs     []*SEQTransaction `json:"txs"`
	BlockID string            `json:"id"`
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

type Header struct {
	Height           uint64  `json:"height"`
	Timestamp        uint64  `json:"timestamp"`
	L1Head           uint64  `json:"l1_head"`
	TransactionsRoot NmtRoot `json:"transactions_root"`
}

func (h *Header) UnmarshalJSON(b []byte) error {
	type Dec struct {
		Height           *uint64  `json:"height"`
		Timestamp        *uint64  `json:"timestamp"`
		L1Head           *uint64  `json:"l1_head"`
		TransactionsRoot *NmtRoot `json:"transactions_root"`
	}

	var dec Dec
	if err := json.Unmarshal(b, &dec); err != nil {
		return err
	}

	if dec.Height == nil {
		return fmt.Errorf("Field height of type Header is required")
	}
	h.Height = *dec.Height

	if dec.Timestamp == nil {
		return fmt.Errorf("Field timestamp of type Header is required")
	}
	h.Timestamp = *dec.Timestamp

	if dec.L1Head == nil {
		return fmt.Errorf("Field l1_head of type Header is required")
	}
	h.L1Head = *dec.L1Head

	if dec.TransactionsRoot == nil {
		return fmt.Errorf("Field transactions_root of type Header is required")
	}
	h.TransactionsRoot = *dec.TransactionsRoot

	return nil
}

func (h *Header) Commit() Commitment {
	return NewRawCommitmentBuilder("BLOCK").
		Uint64Field("height", h.Height).
		Uint64Field("timestamp", h.Timestamp).
		Uint64Field("l1_head", h.L1Head).
		Field("transactions_root", h.TransactionsRoot.Commit()).
		Finalize()
}

type NmtRoot struct {
	Root Bytes `json:"root"`
}

func (r *NmtRoot) UnmarshalJSON(b []byte) error {
	// Parse using pointers so we can distinguish between missing and default fields.
	type Dec struct {
		Root *Bytes `json:"root"`
	}

	var dec Dec
	if err := json.Unmarshal(b, &dec); err != nil {
		return err
	}

	if dec.Root == nil {
		return fmt.Errorf("Field root of type NmtRoot is required")
	}
	r.Root = *dec.Root

	return nil
}

func (r *NmtRoot) Commit() Commitment {
	return NewRawCommitmentBuilder("NMTROOT").
		VarSizeField("root", r.Root).
		Finalize()
}

type Bytes []byte

type BlockInfo struct {
	BlockID   string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	L1Head    uint64 `json:"l1_head"`
	Height    uint64 `json:"height"`
}

type BlockHeadersResponse struct {
	From   uint64      `json:"from"`
	Blocks []BlockInfo `json:"blocks"`
	Prev   BlockInfo   `json:"prev"`
	Next   BlockInfo   `json:"next"`
}

type GetBlockHeadersIDArgs struct {
	ID           string `json:"id"`
	EndTimeStamp int64  `json:"end_timestamp"`
}

type GetBlockHeadersByHeightArgs struct {
	Height       uint64 `json:"height"`
	EndTimeStamp int64  `json:"end_timestamp"`
}

type GetBlockHeadersByStartTimeStampArgs struct {
	StartTimeStamp int64 `json:"start_timestamp"`
	EndTimeStamp   int64 `json:"end_timestamp"`
}

type GetBlockCommitmentArgs struct {
	First         uint64 `json:"first"`
	CurrentHeight uint64 `json:"current_height"`
	MaxBlocks     int    `json:"max_blocks"`
}

type SequencerWarpBlockResponse struct {
	Blocks []SequencerWarpBlock `json:"blocks"`
}

type SequencerWarpBlock struct {
	BlockID    string   `json:"id"`
	Timestamp  int64    `json:"timestamp"`
	L1Head     uint64   `json:"l1_head"`
	Height     *big.Int `json:"height"`
	BlockRoot  *big.Int `json:"root"`
	ParentRoot *big.Int `json:"parent"`
}

type GetBlockTransactionsArgs struct {
	ID string `json:"block_id"`
}

type TransactionResponse struct {
	Txs     []*chain.Transaction `json:"txs"`
	BlockID string               `json:"id"`
}

type GetBlockTransactionsByNamespaceArgs struct {
	Height    uint64 `json:"height"`
	Namespace string `json:"namespace"`
}

type RegistryReply struct {
	Namespaces [][]byte `json:"namespaces"`
}

type EpochExitsArgs struct {
	Epoch uint64 `json:"epoch"`
}

type EpochExitsReply struct {
	Info *hactions.EpochExitInfo `json:"info"`
}

type GetRollupInfoArgs struct {
	Namespace []byte `json:"namespace"`
}

type GetRollupInfoReply struct {
	Info hactions.RollupInfo `json:"rollupInfo"`
}

type GetBuilderArgs struct {
	Epoch uint64 `json:"epoch"`
}

type GetBuilderReply struct {
	BuilderPubKey []byte `json:"builderPubKey"`
}

type GetAllRollupInfoReply struct {
	// epoch -> a list of rollup registration info
	Info map[uint64][]hactions.RollupInfo `json:"info"`
}

type GetRollupsInfoAtEpochArgs struct {
	Epoch uint64 `json:"epoch"`
}

type GetRollupsInfoAtEpochReply struct {
	Rollups []*hactions.RollupInfo `json:"rollups"`
}

type GetCertByChunkIDArgs struct {
	ChunkID ids.ID `json:"chunkID"`
}

type GetCertByChunkIDReply struct {
	Cert *DACertInfo `json:"cert"`
}

type GetCertByChainInfoArgs struct {
	ChainID     string `json:"chainID"`
	BlockNumber uint64 `json:"blockNumber"`
}

type GetCertByChainInfoReply struct {
	Cert *DACertInfo `json:"cert"`
}

type GetCertsByToBNonceArgs struct {
	ToBNonce uint64 `json:"tobNonce"`
}

type GetCertsByToBNonceReply struct {
	Certs []*DACertInfo `json:"certs"`
}

type GetLowestToBNonceAtEpochArgs struct {
	Epoch uint64 `json:"epoch"`
}

type GetLowestToBNonceAtEpochReply struct {
	ToBNonce uint64 `json:"tobNonce"`
}
