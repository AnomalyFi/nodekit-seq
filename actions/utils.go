package actions

import (
	"math/big"

	"github.com/AnomalyFi/hypersdk/codec"
	"github.com/ava-labs/coreth/accounts/abi/bind"
)

func ContainsAddress(addrs []codec.Address, addr codec.Address) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}

// GnarkPrecompileInputs is an auto generated low-level Go binding around an user-defined struct.
type GnarkPrecompileInputs struct {
	ProgramVKeyHash [32]byte
	PublicValues    []byte
	ProofBytes      []byte
	ProgramVKey     []byte
}

// GnarkPreCompileMetaData contains all meta data concerning the SolGen contract.
var GnarkPreCompileMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"programVKeyHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"publicValues\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proofBytes\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"programVKey\",\"type\":\"bytes\"}],\"internalType\":\"structSolGen.gnarkPrecompileInputs\",\"name\":\"inputs\",\"type\":\"tuple\"}],\"name\":\"gnarkPrecompile\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

var GnarkPreCompileABI, _ = GnarkPreCompileMetaData.GetAbi()

var mask = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 253), big.NewInt(1))
