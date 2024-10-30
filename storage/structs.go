package storage

import (
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"

	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/ava-labs/avalanchego/ids"
)

// TODO need to fix this to be a rollup exiting the epoch
// The way the mapping needs to work is that we have a list of all the epochs
// and then we are updating the epoch's when a rollup opts out we can append the namespace there.
type EpochInfo struct {
	Epoch     uint64 `json:"epoch"`
	Namespace []byte `json:"namespace"`
}

type EpochExitInfo struct {
	Exits []*EpochInfo `json:"exits"`
}

func (e *EpochExitInfo) Marshal(p *codec.Packer) error {
	p.PackInt(len(e.Exits))
	for _, exit := range e.Exits {
		p.PackBytes(exit.Namespace)
		p.PackUint64(exit.Epoch)
	}
	return p.Err()
}

func UnmarshalEpochExitInfo(p *codec.Packer) (*EpochExitInfo, error) {
	count := p.UnpackInt(true)
	exits := make([]*EpochInfo, count)
	for i := 0; i < count; i++ {
		ret := new(EpochInfo)
		p.UnpackBytes(32, false, &ret.Namespace)
		//p.UnpackUint64(&ret.Epoch)
		ret.Epoch = p.UnpackUint64(true)

		if err := p.Err(); err != nil {
			return nil, err
		}

		exits[i] = ret
	}
	return &EpochExitInfo{Exits: exits}, p.Err()
}

func NewEpochInfo(namespace []byte, epoch uint64) *EpochInfo {
	return &EpochInfo{
		Epoch:     epoch,
		Namespace: namespace,
	}
}

func (a *EpochInfo) ID() ids.ID {
	return utils.ToID([]byte(a.Namespace))
}

func (a *EpochInfo) Size() int {
	return consts.Uint64Len + codec.BytesLen(a.Namespace)
}

func (a *EpochInfo) Marshal(p *codec.Packer) error {
	p.PackBytes(a.Namespace)
	p.PackUint64(a.Epoch)

	if err := p.Err(); err != nil {
		return err
	}

	return nil
}

func UnmarshalEpochInfo(p *codec.Packer) (*EpochInfo, error) {
	ret := new(EpochInfo)

	p.UnpackBytes(32, false, &ret.Namespace) // TODO: set limit for the length of namespace bytes
	ret.Epoch = p.UnpackUint64(true)

	if err := p.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}
