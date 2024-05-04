package rpc

import (
	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/nodekit-seq/actions"
)

const seqWasmMode byte = 0

func PackSEQWasmTxs(b *chain.StatelessBlock) ([]byte, error) {
	var seqWasmTxs []*chain.Transaction

	for _, tx := range b.Txs {
		switch tx.Action.(type) {
		case *actions.Transact:
			seqWasmTxs = append(seqWasmTxs, tx)
		case *actions.Deploy:
			seqWasmTxs = append(seqWasmTxs, tx)
		}
	}
	size := codec.CummSize(seqWasmTxs)
	p := codec.NewWriter(size, consts.NetworkSizeLimit)
	p.PackInt(len(b.Txs))
	for _, tx := range b.Txs {
		if err := tx.Marshal(p); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func UnpackSEQWasmTxs(b []byte, parser chain.Parser) ([]*chain.Transaction, error) {
	p := codec.NewReader(b, consts.NetworkSizeLimit)
	numTxs := p.UnpackInt(true)
	txs := make([]*chain.Transaction, numTxs)
	actionRegistry, authRegistry := parser.Registry()
	for i := 0; i < numTxs; i++ {
		tx, err := chain.UnmarshalTx(p, actionRegistry, authRegistry)
		if err != nil {
			return nil, err
		}
		txs[i] = tx
	}
	return txs, nil
}
