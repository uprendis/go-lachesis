package genesis

import (
	"math/big"

	"github.com/Fantom-foundation/go-lachesis/crypto"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

// FakeValidators returns validators accounts for fakenet
func FakeValidators(count int, balance *big.Int, stake *big.Int) VAccounts {
	accs := make(Accounts, count)
	validators := make(GValidators, 0, count)

	for i := 1; i <= count; i++ {
		key := crypto.FakeKey(i)
		addr := crypto.PubkeyToAddress(key.PublicKey)
		accs[addr] = Account{
			Balance:    balance,
			PrivateKey: key,
		}
		stakerID := idx.StakerID(i)
		validators = append(validators, GenesisValidator{
			ID:      stakerID,
			Address: addr,
			Stake:   stake,
		})
	}

	return VAccounts{Accounts: accs, Validators: validators, SfcContractAdmin: validators[0].Address}
}
