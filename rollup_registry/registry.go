package rollupregistry

import (
	"maps"
	"sync"

	hactions "github.com/AnomalyFi/hypersdk/actions"
	"github.com/ava-labs/avalanchego/ids"
)

type RollupRegistryOnlyRead interface {
	RollupsValidAtEpoch(epoch uint64) []*hactions.RollupInfo
}

type RollupRegistryAllPerm interface {
	RollupRegistryOnlyRead
	Update(currentEpoch uint64, rollups []*hactions.RollupInfo)
}

var _ RollupRegistryAllPerm = (*RollupRegistry)(nil)

type RollupRegistry struct {
	rollups  map[ids.ID]*hactions.RollupInfo
	rollupsL sync.RWMutex
}

func NewRollupRegistr() *RollupRegistry {
	return &RollupRegistry{
		rollups: make(map[ids.ID]*hactions.RollupInfo),
	}
}

func (r *RollupRegistry) RollupsValidAtEpoch(epoch uint64) []*hactions.RollupInfo {
	r.rollupsL.RLock()
	defer r.rollupsL.RUnlock()

	ret := make([]*hactions.RollupInfo, 0)
	for _, rollup := range r.rollups {
		if !rollup.ValidAtEpoch(epoch) {
			continue
		}

		ret = append(ret, rollup)
	}

	return ret
}

func (r *RollupRegistry) Update(currentEpoch uint64, rollups []*hactions.RollupInfo) {
	r.rollupsL.Lock()
	defer r.rollupsL.Unlock()

	for _, rollup := range rollups {
		r.rollups[rollup.ID()] = rollup
	}

	maps.DeleteFunc(r.rollups, func(_ ids.ID, rollup *hactions.RollupInfo) bool {
		return rollup.Exited(currentEpoch)
	})
}
