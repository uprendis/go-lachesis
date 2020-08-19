package genesis

import (
	"time"

	"github.com/Fantom-foundation/go-lachesis/inter"
)

var (
	genesisTime = inter.Timestamp(1597727000 * time.Second)
)

type Genesis struct {
	Validators Validators
	Time       inter.Timestamp
	ExtraData  []byte
}

// FakeGenesis generates fake genesis with n-nodes.
func FakeGenesis(accs Validators) Genesis {
	return Genesis{
		Validators: accs,
		Time:       genesisTime,
		ExtraData:  []byte("experimental benchopera"),
	}
}
