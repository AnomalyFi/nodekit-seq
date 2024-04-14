package actions

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/anomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
)

var _ chain.Action = (*Deploy)(nil)

type Deploy struct {
	ContractID   ids.ID `json:"contractID"`
	ContractCode []byte `json:"contractCode"`
}

func (*Deploy) GetTypeID() uint8 {
	return DeployID
}

func (d *Deploy) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.ContractKey(d.ContractID)): state.Read | state.Write,
	}
}

func (*Deploy) StateKeysMaxChunks() []uint16 {
	return []uint16{DeployMaxChunks}
}

func (*Deploy) OutputsWarpMessage() bool {
	return false
}

func (d *Deploy) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
	_ bool) (bool, uint64, []byte /*output*/, *warp.UnsignedMessage, error) {
	// @todo add authentication for this action type. only nodekit-approved address should be able to call this action
	code := d.ContractCode
	// units := uint64(codec.BytesLen(code))
	code2, err := storage.GetContract(ctx, mu, d.ContractID)
	if err != nil {
		return false, 10_000, utils.ErrBytes(errors.New("cant find contract")), nil, nil
	}
	code = append(code2, code...)
	if err := storage.SetContract(ctx, mu, d.ContractID, code); err != nil {
		return false, 10_000, utils.ErrBytes(fmt.Errorf("cant set contract: %s", err)), nil, nil
	}
	return true, 10_000, nil, nil, nil
}

func (*Deploy) MaxComputeUnits(chain.Rules) uint64 {
	return DeployMaxComputeUnits
}

func (d *Deploy) Marshal(p *codec.Packer) {
	p.PackID(d.ContractID)
	p.PackBytes(d.ContractCode)
}

func UnmarshalDeploy(p *codec.Packer, _ *warp.Message) (chain.Action, error) {
	var deploy Deploy
	p.UnpackID(true, &deploy.ContractID)
	p.UnpackBytes(int(math.MaxUint32), true, &deploy.ContractCode)
	return &deploy, p.Err()
}

func (d *Deploy) Size() int {
	return codec.BytesLen(d.ContractCode) + consts.IDLen
}

func (*Deploy) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (d *Deploy) Chunks() uint16 {
	return uint16(len(d.ContractCode) / 64) // size of chunk = 64
}
