package storage

import (
	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/ava-labs/avalanchego/ids"
)

func PackNamespaces(namespaces [][]byte) ([]byte, error) {
	return hactions.PackNamespaces(namespaces)
}

func UnpackNamespaces(raw []byte) ([][]byte, error) {
	return hactions.UnpackNamespaces(raw)
}

func PackEpochs(epochs []uint64) ([]byte, error) {
	p := codec.NewWriter(len(epochs)*8, consts.NetworkSizeLimit)
	p.PackInt(len(epochs))
	for _, e := range epochs {
		p.PackUint64(e)
	}
	return p.Bytes(), p.Err()
}

func UnpackEpochs(raw []byte) ([]uint64, error) {
	p := codec.NewReader(raw, consts.NetworkSizeLimit)
	eLen := p.UnpackInt(false)
	epochs := make([]uint64, 0, eLen)
	for i := 0; i < eLen; i++ {
		e := p.UnpackUint64(true)
		epochs = append(epochs, e)
	}

	return epochs, p.Err()
}

func PackIDs(ids []ids.ID) ([]byte, error) {
	p := codec.NewWriter(1024, consts.NetworkSizeLimit)
	p.PackInt(len(ids))
	for _, id := range ids {
		p.PackID(id)
	}

	return p.Bytes(), p.Err()
}

func UnpackIDs(raw []byte) ([]ids.ID, error) {
	p := codec.NewReader(raw, consts.NetworkSizeLimit)
	numIDs := p.UnpackInt(false)
	ret := make([]ids.ID, numIDs)
	for _, id := range ret {
		p.UnpackID(true, &id)
		if err := p.Err(); err != nil {
			return nil, err
		}
	}
	return ret, nil
}
