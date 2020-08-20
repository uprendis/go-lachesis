package benchopera

import (
	"github.com/Fantom-foundation/go-lachesis/benchopera/genesis"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

const (
	FakeNetworkID uint64 = 0xefa3
)

// DagConfig of Lachesis DAG (directed acyclic graph).
type DagConfig struct {
	MaxParents     int `json:"maxParents"`
	MaxFreeParents int `json:"maxFreeParents"` // maximum number of parents with no gas cost

	MaxEpochBlocks idx.Block `json:"maxEpochBlocks"`
}

// Config describes benchopera net.
type Config struct {
	Name      string
	NetworkID uint64

	Genesis genesis.Genesis

	// Graph options
	Dag DagConfig
}

func FakeNetConfig(accs genesis.Validators) Config {
	return Config{
		Name:      "fake",
		NetworkID: FakeNetworkID,
		Genesis:   genesis.FakeGenesis(accs),
		Dag:       FakeNetDagConfig(),
	}
}

func DefaultDagConfig() DagConfig {
	return DagConfig{
		MaxParents:     10,
		MaxFreeParents: 3,
		MaxEpochBlocks: 10,
	}
}

func FakeNetDagConfig() DagConfig {
	return DefaultDagConfig()
}
