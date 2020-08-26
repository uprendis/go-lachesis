package adapters

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type EvmToVmi struct {
	*vm.EVM
}

func (vm *EvmToVmi) GetStateDB() vm.StateDB {
	return vm.StateDB
}

func (vm *EvmToVmi) ContractCreation() common.Address {
	return vm.Context.Origin
}
