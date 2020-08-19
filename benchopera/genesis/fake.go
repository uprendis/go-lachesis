package genesis

import (
	"crypto/ecdsa"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"math/rand"
)

// FakeKey gets n-th fake private key.
func FakeKey(n int) *ecdsa.PrivateKey {
	reader := rand.New(rand.NewSource(int64(n)))

	key, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic(err)
	}

	return key
}

// FakeValidators returns validators accounts for fakenet
func FakeValidators(count int, stake *big.Int) Validators {
	validators := make(Validators, 0, count)

	for i := 1; i <= count; i++ {
		key := FakeKey(i)
		validatorID := idx.ValidatorID(i)
		validators = append(validators, Validator{
			ID:     validatorID,
			PubKey: crypto.FromECDSAPub(&key.PublicKey),
			Stake:  stake,
		})
	}

	return validators
}
