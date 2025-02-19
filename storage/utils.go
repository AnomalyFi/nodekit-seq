package storage

import (
	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
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
