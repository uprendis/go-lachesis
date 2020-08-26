package evmcore

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/Fantom-foundation/go-lachesis/utils/adapters"
)

// VMI is a VM interface
type VMI interface {
	// Create creates a new contract using code as deployment code.
	Create(caller vm.ContractRef, code []byte, gas uint64, value *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error)
	// Call executes the contract associated with the addr with the given input as
	// parameters. It also handles any necessary value transfer required and takes
	// the necessary steps to create accounts and reverses the state in case of an
	// execution error or failed value transfer.
	Call(caller vm.ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)

	// GetStateDB returns StateDB which gives access to the underlying state
	GetStateDB() vm.StateDB

	// ContractCreation returns an address of a created contract (if any)
	ContractCreation() common.Address

	// Cancel cancels any running VM operation. This may be called concurrently and
	// it's safe to be called multiple times.
	Cancel()
	// Cancelled returns true if Cancel has been called
	Cancelled() bool
}

// NewDefaultVM creates a default VM
func NewDefaultVM(msg Message, header *EvmHeader, bc DummyChain, statedb vm.StateDB) VMI {
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, nil)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	evm := vm.NewEVM(context, statedb, params.AllEthashProtocolChanges, vm.Config{})
	return &adapters.EvmToVmi{evm}
}
