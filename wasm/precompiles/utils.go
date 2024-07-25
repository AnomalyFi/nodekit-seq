package precompiles

import (
	"math/big"

	"github.com/consensys/gnark/frontend"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type SP1Circuit struct {
	VkeyHash             frontend.Variable `gnark:",public"`
	CommitedValuesDigest frontend.Variable `gnark:",public"`
	Vars                 []frontend.Variable
	Felts                []babybearVariable
	Exts                 []babybearExtensionVariable
}

func (*SP1Circuit) Define(frontend.API) error {
	return nil
}

type babybearVariable struct {
	Value  frontend.Variable
	NbBits uint
}

type babybearExtensionVariable struct {
	Value [4]babybearVariable
}

// GnarkPrecompileInputs is an auto generated low-level Go binding around an user-defined struct.
type GnarkPrecompileInputs struct {
	ProgramVKeyHash []byte
	PublicValues    []byte
	ProofBytes      []byte
	ProgramVKey     []byte
}

// GnarkPreCompileMetaData contains all meta data concerning the SolGen contract.
var GnarkPreCompileMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"bytes\",\"name\":\"programVKeyHash\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"publicValues\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"proofBytes\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"programVKey\",\"type\":\"bytes\"}],\"internalType\":\"structSolGen.gnarkPrecompileInputs\",\"name\":\"inputs\",\"type\":\"tuple\"}],\"name\":\"gnarkPrecompile\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

var GnarkPreCompileABI, _ = GnarkPreCompileMetaData.GetAbi()

var mask = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 253), big.NewInt(1))
