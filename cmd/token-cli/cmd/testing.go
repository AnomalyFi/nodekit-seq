package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"

	"github.com/AnomalyFi/hypersdk/utils"
	"github.com/AnomalyFi/nodekit-seq/actions"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/spf13/cobra"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// BinaryMerkleProof is an auto generated low-level Go binding around an user-defined struct.
type BinaryMerkleProof struct {
	SideNodes [][32]byte
	Key       *big.Int
	NumLeaves *big.Int
}

// DataRootTuple is an auto generated low-level Go binding around an user-defined struct.
type DataRootTuple struct {
	Height   *big.Int
	DataRoot [32]byte
}

// Signature is an auto generated low-level Go binding around an user-defined struct.
type Signature struct {
	V uint8
	R [32]byte
	S [32]byte
}

// SubmitDataRootTupleRootInput is an auto generated low-level Go binding around an user-defined struct.
type SubmitDataRootTupleRootInput struct {
	NewNonce            *big.Int
	ValidatorSetNonce   *big.Int
	DataRootTupleRoot   [32]byte
	CurrentValidatorSet []Validator
	Sigs                []Signature
}

// UpdateValidatorSetInput is an auto generated low-level Go binding around an user-defined struct.
type UpdateValidatorSetInput struct {
	NewNonce            *big.Int
	OldNonce            *big.Int
	NewPowerThreshold   *big.Int
	NewValidatorSetHash [32]byte
	CurrentValidatorSet []Validator
	Sigs                []Signature
}

// Validator is an auto generated low-level Go binding around an user-defined struct.
type Validator struct {
	Addr  common.Address
	Power *big.Int
}

// VerifyAttestationInput is an auto generated low-level Go binding around an user-defined struct.
type VerifyAttestationInput struct {
	TupleRootNonce *big.Int
	Tuple          DataRootTuple
	Proof          BinaryMerkleProof
}

// InputsInitializerInput is an auto generated low-level Go binding around an user-defined struct.
type InputsInitializerInput struct {
	Nonce                  *big.Int
	PowerThreshold         *big.Int
	ValidatorSetCheckPoint [32]byte
}

// MainMetaData contains all meta data concerning the Main contract.
var MainMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"nonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"power_threshold\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"validator_set_check_point\",\"type\":\"bytes32\"}],\"internalType\":\"structinputs.InitializerInput\",\"name\":\"_i\",\"type\":\"tuple\"}],\"name\":\"dummyInitializerInput\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"_newNonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_validatorSetNonce\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_dataRootTupleRoot\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"power\",\"type\":\"uint256\"}],\"internalType\":\"structinputs.Validator[]\",\"name\":\"_currentValidatorSet\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structinputs.Signature[]\",\"name\":\"_sigs\",\"type\":\"tuple[]\"}],\"internalType\":\"structinputs.SubmitDataRootTupleRootInput\",\"name\":\"_s\",\"type\":\"tuple\"}],\"name\":\"dummySubmitDataRootTupleRootInput\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"_newNonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_oldNonce\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_newPowerThreshold\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"_newValidatorSetHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"power\",\"type\":\"uint256\"}],\"internalType\":\"structinputs.Validator[]\",\"name\":\"_currentValidatorSet\",\"type\":\"tuple[]\"},{\"components\":[{\"internalType\":\"uint8\",\"name\":\"v\",\"type\":\"uint8\"},{\"internalType\":\"bytes32\",\"name\":\"r\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"internalType\":\"structinputs.Signature[]\",\"name\":\"_sigs\",\"type\":\"tuple[]\"}],\"internalType\":\"structinputs.UpdateValidatorSetInput\",\"name\":\"_u\",\"type\":\"tuple\"}],\"name\":\"dummyUpdateValidatorSet\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"_tupleRootNonce\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"height\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"dataRoot\",\"type\":\"bytes32\"}],\"internalType\":\"structinputs.DataRootTuple\",\"name\":\"_tuple\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bytes32[]\",\"name\":\"sideNodes\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"key\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"numLeaves\",\"type\":\"uint256\"}],\"internalType\":\"structinputs.BinaryMerkleProof\",\"name\":\"_proof\",\"type\":\"tuple\"}],\"internalType\":\"structinputs.VerifyAttestationInput\",\"name\":\"_v\",\"type\":\"tuple\"}],\"name\":\"dummyVerifyAttestationInput\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// MainABI is the input ABI used to generate the binding from.
// Deprecated: Use MainMetaData.ABI instead.
var MainABI, _ = MainMetaData.GetAbi() // modified

var testingCmd = &cobra.Command{
	Use: "testing",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var deployTCmd = &cobra.Command{
	Use: "deploy",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		codeLocation := "/home/manojkgorle/nodekit/seq-wasm/blobstream-contracts-rust/target/wasm32-unknown-unknown/release/blobstream_contracts_rust.wasm"
		code, err := ioutil.ReadFile(codeLocation)
		if err != nil {
			return err
		}

		contractAddress, err := handler.Root().PromptAsset("contract Address", false)
		if err != nil {
			return err
		}
		chunkSize := 100 * 1024
		for i := 0; i < len(code); i += chunkSize {
			end := i + chunkSize
			if end > len(code) {
				end = len(code)
			}
			result, txID, _ := sendAndWait(ctx, nil, &actions.Deploy{
				ContractID:   contractAddress,
				ContractCode: code[i:end],
			}, cli, scli, tcli, factory, true)
			utils.Outf("{{green}} result: %t{{red}} txID: %s\n", result, txID)
		}
		return err
	},
}

var initializeCmd = &cobra.Command{
	Use: "initialize",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		contractAddress, err := handler.Root().PromptAsset("contract Address", false)
		if err != nil {
			return err
		}
		initializerStruct := InputsInitializerInput{
			Nonce:                  big.NewInt(1),
			PowerThreshold:         big.NewInt(3333),
			ValidatorSetCheckPoint: [32]byte(common.Hex2BytesFixed("4a5cc92ce4a0fb368c83da44ea489e4b908ce75bdc460c31c662f35fd3911ff1", 32)),
		}
		// Encode the parameters using the ABI packer
		packed, err := abi.ABI.Pack(*MainABI, "dummyInitializerInput", initializerStruct)
		if err != nil {
			return err
		}
		input := packed[4:]
		result, txID, _ := sendAndWait(ctx, nil, &actions.Transact{
			FunctionName:    "initializer",
			ContractAddress: contractAddress,
			Input:           input,
		}, cli, scli, tcli, factory, true)
		utils.Outf("{{green}} result: %t{{red}} txID: %s\n", result, txID)
		return err
	},
}

var sdrtrCmd = &cobra.Command{
	Use: "submit",
	RunE: func(*cobra.Command, []string) error {
		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}
		contractAddress, err := handler.Root().PromptAsset("contract Address", false)
		if err != nil {
			return err
		}
		sdrtri := SubmitDataRootTupleRootInput{
			NewNonce:            big.NewInt(2),
			ValidatorSetNonce:   big.NewInt(1),
			DataRootTupleRoot:   [32]byte(common.Hex2BytesFixed("0de92bac0b356560d821f8e7b6f5c9fe4f3f88f6c822283efd7ab51ad56a640e", 32)),
			CurrentValidatorSet: []Validator{{common.HexToAddress("9c2B12b5a07FC6D719Ed7646e5041A7E85758329"), big.NewInt(5000)}},
			Sigs:                []Signature{{V: 28, R: [32]byte(common.Hex2BytesFixed("f48f949c827fb5a0db3bf416ea657d2750eeadb7b6906c6fb857d2fd1dd57181", 32)), S: [32]byte(common.Hex2BytesFixed("46ae888d1453fd5693b0148cecf0368b42552e597a3b628456946cf63b627b04", 32))}},
		}
		packed, err := abi.ABI.Pack(*MainABI, "dummySubmitDataRootTupleRootInput", sdrtri)
		if err != nil {
			fmt.Print(err)
		}
		input := packed[4:]
		result, txID, _ := sendAndWait(ctx, nil, &actions.Transact{
			FunctionName:    "submit_data_root_tuple_root",
			ContractAddress: contractAddress,
			Input:           input,
		}, cli, scli, tcli, factory, true)
		utils.Outf("{{green}} result: %t{{red}} txID: %s\n", result, txID)
		return err
	},
}
