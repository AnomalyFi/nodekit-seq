package actions

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
)

var _ chain.Action = (*Deploy)(nil)

type Deploy struct {
	ContractID   ids.ID `json:"contractID"`
	ContractCode []byte `json:"contractCode"`
}

func (*Deploy) GetTypeID() uint8 {
	return deployID
}

func (d *Deploy) StateKeys(actor codec.Address, txID ids.ID) state.Keys {
	return state.Keys{
		string(storage.ContractKey(d.ContractID)): state.All,
	}
}

func (*Deploy) StateKeysMaxChunks() []uint16 {
	return []uint16{DeployMaxChunks}
}

func (d *Deploy) Execute(
	ctx context.Context,
	rules chain.Rules,
	mu state.Mutable,
	_ int64,
	actor codec.Address,
	txID ids.ID,
) ([][]byte, error) {
	whitelistedAddressesB, _ := rules.FetchCustom("whitelisted.Addresses")
	whitelistedAddresses := whitelistedAddressesB.([]codec.Address)
	if !ContainsAddress(whitelistedAddresses, actor) {
		return nil, ErrNotWhiteListed
	}
	code := d.ContractCode
	// units := uint64(codec.BytesLen(code))
	code2, err := storage.GetContract(ctx, mu, d.ContractID)
	if err != nil {
		return nil, errors.New("cant find contract")
	}
	code = append(code2, code...)
	if err := storage.SetContract(ctx, mu, d.ContractID, code); err != nil {
		return nil, fmt.Errorf("cant set contract: %s", err)
	}
	return nil, nil
}

func (*Deploy) ComputeUnits(codec.Address, chain.Rules) uint64 {
	return DeployComputeUnits
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
	return codec.BytesLen(d.ContractCode) + ids.IDLen
}

func (*Deploy) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}

func (*Deploy) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*Deploy) UseFeeMarket() bool {
	return false
}
