package actions

import (
	"context"

	"github.com/AnomalyFi/hypersdk/chain"
	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/AnomalyFi/hypersdk/consts"
	"github.com/AnomalyFi/hypersdk/state"
	"github.com/AnomalyFi/nodekit-seq/storage"
	"github.com/ava-labs/avalanchego/ids"
)

var _ chain.Action = (*Oracle)(nil)

type Oracle struct {
	RelayerIDs    []int    `json:"relayer_ids"`
	UnitGasPrices []uint64 `json:"unit_gas_prices"`
}

func (*Oracle) GetTypeID() uint8 {
	return oracleID
}

func (o *Oracle) StateKeys(_ codec.Address, actionID ids.ID) state.Keys {
	var keys state.Keys
	for _, relayerID := range o.RelayerIDs {
		keys.Add(string(storage.RelayerGasPriceKey(relayerID)), state.Allocate|state.Write)
		keys.Add(string(storage.RelayerGasPriceUpdateTimeStampKey(relayerID)), state.Allocate|state.Write)
	}
	return keys
}

func (o *Oracle) StateKeysMaxChunks() []uint16 {
	var chunks []uint16
	for range o.RelayerIDs {
		chunks = append(chunks, storage.RelayerGasChunks)
		chunks = append(chunks, storage.RelayerGasTimeStampChunks)
	}
	return chunks
}

func (o *Oracle) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	timeStamp int64,
	_ codec.Address,
	_ ids.ID,
) ([][]byte, error) {
	// @todo changes in genesis and rules for getting config.
	// @todo verify the actor is the whitelisted authority?
	if len(o.RelayerIDs) != len(o.UnitGasPrices) {
		return nil, ErrRelayerIDsUnitGasPricesMismatch
	}
	for i := range o.RelayerIDs {
		if err := storage.StoreRelayerGasPrice(ctx, mu, o.RelayerIDs[i], o.UnitGasPrices[i]); err != nil {
			return nil, err
		}
		if err := storage.StoreRelayerGasPriceUpdateTimeStamp(ctx, mu, o.RelayerIDs[i], timeStamp); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// TODO: tune this to reflect the spam prevention mechanism.
func (*Oracle) ComputeUnits(codec.Address, chain.Rules) uint64 {
	// @todo verify the actor is the whitelisted authority?
	// verify if relayerIDs len match unitGasPrices len
	return OracleComputeUnits
}

func (*Oracle) Size() int {
	return consts.IntLen + consts.Uint64Len
}

func (o *Oracle) Marshal(p *codec.Packer) {
	p.PackInt(len(o.RelayerIDs))
	for _, relayerID := range o.RelayerIDs {
		p.PackInt(relayerID)
	}
	p.PackInt(len(o.UnitGasPrices))
	for _, unitGasPrice := range o.UnitGasPrices {
		p.PackUint64(unitGasPrice)
	}
}

func UnmarshalOracle(p *codec.Packer) (chain.Action, error) {
	var oracle Oracle
	relayerIDsLen := p.UnpackInt(true)
	for i := 0; i < relayerIDsLen; i++ {
		oracle.RelayerIDs = append(oracle.RelayerIDs, p.UnpackInt(true))
	}
	unitGasPricesLen := p.UnpackInt(true)
	for i := 0; i < unitGasPricesLen; i++ {
		oracle.UnitGasPrices = append(oracle.UnitGasPrices, p.UnpackUint64(true))
	}
	return &oracle, nil
}

func (*Oracle) ValidRange(chain.Rules) (int64, int64) {
	return -1, -1
}

func (*Oracle) NMTNamespace() []byte {
	return DefaultNMTNamespace
}

func (*Oracle) UseFeeMarket() bool {
	return false
}
