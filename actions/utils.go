package actions

import (
	"math/big"

	"github.com/AnomalyFi/hypersdk/codec"

	"github.com/consensys/gnark/frontend"
	gtype "github.com/succinctlabs/gnark-plonky2-verifier/types"
	"github.com/succinctlabs/gnark-plonky2-verifier/variables"
)

func ContainsAddress(addrs []codec.Address, addr codec.Address) bool {
	for _, a := range addrs {
		if a == addr {
			return true
		}
	}
	return false
}

// GnarkPrecompileGnarkPrecompileInputs is an auto generated low-level Go binding around an user-defined struct.
type GnarkPrecompileInputs struct {
	Input            []byte
	Output           []byte
	Proof            []byte
	FunctionIdBigInt *big.Int
}

var mask = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 253), big.NewInt(1))

// can't load this from the package. why?? plonky2x is not a go module

// We assume that the publicInputs have 64 bytes
// publicInputs[0:32] is a big-endian representation of a SHA256 hash that has been truncated to 253 bits.
// Note that this truncation happens in the `WrappedCircuit` when computing the `input_hash`
// The reason for truncation is that we only want 1 public input on-chain for the input hash
// to save on gas costs
type Plonky2xVerifierCircuit struct {
	// A digest of the plonky2x circuit that is being verified.
	VerifierDigest frontend.Variable `gnark:"verifierDigest,public"`

	// The input hash is the hash of all onchain inputs into the function.
	InputHash frontend.Variable `gnark:"inputHash,public"`

	// The output hash is the hash of all outputs from the function.
	OutputHash frontend.Variable `gnark:"outputHash,public"`

	// Private inputs to the circuit
	ProofWithPis variables.ProofWithPublicInputs
	VerifierData variables.VerifierOnlyCircuitData

	// Circuit configuration that is not part of the circuit itself.
	CommonCircuitData gtype.CommonCircuitData `gnark:"-"`
}

func (c *Plonky2xVerifierCircuit) Define(api frontend.API) error { return nil }
