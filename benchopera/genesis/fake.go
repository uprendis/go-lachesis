package genesis

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"math/big"

	"github.com/Fantom-foundation/go-lachesis/crypto"

)

// FakeValidators returns validators accounts for fakenet
func FakeValidators(count int, stake *big.Int) Validators {
	validators := make(Validators, 0, count)

	for i := 1; i <= count; i++ {
		key := crypto.FakeKey(i)
		addr := crypto.PubkeyToAddress(key.PublicKey)
		validatorID := idx.ValidatorID(i)
		validators = append(validators, Validator{
			ID:      validatorID,
			Address: addr,
			Stake:   stake,
		})
	}

	return validators
}
