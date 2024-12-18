package rollupregistry

import (
	"slices"
	"sync"

	hactions "github.com/AnomalyFi/hypersdk/actions"
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
	rollups  []*hactions.RollupInfo
	rollupsL sync.RWMutex
}

func NewRollupRegistr() *RollupRegistry {
	return &RollupRegistry{
		rollups: make([]*hactions.RollupInfo, 0),
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

	r.rollups = append(r.rollups, rollups...)
	// remove exited rollups
	r.rollups = slices.DeleteFunc(r.rollups, func(rollup *hactions.RollupInfo) bool {
		return rollup.Exited(currentEpoch)
	})
}
