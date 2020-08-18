package network

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"math/big"
	"time"

	ethparams "github.com/ethereum/go-ethereum/params"

	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/network/genesis"
	"github.com/Fantom-foundation/go-lachesis/network/params"
)

const (
	MainNetworkID uint64 = 0xfa
	TestNetworkID uint64 = 0xfa2
	FakeNetworkID uint64 = 0xfa3
)

var (
	// PercentUnit is used to define ratios with integers, it's 1.0
	PercentUnit = big.NewInt(1e6)
)

// GasPowerConfig defines gas power rules in the consensus.
type GasPowerConfig struct {
	InitialAllocPerSec uint64          `json:"initialAllocPerSec"`
	MaxAllocPerSec     uint64          `json:"maxAllocPerSec"`
	MinAllocPerSec     uint64          `json:"minAllocPerSec"`
	MaxAllocPeriod     inter.Timestamp `json:"maxAllocPeriod"`
	StartupAllocPeriod inter.Timestamp `json:"startupAllocPeriod"`
	MinStartupGas      uint64          `json:"minStartupGas"`
}

// DagConfig of Lachesis DAG (directed acyclic graph).
type DagConfig struct {
	MaxParents     int `json:"maxParents"`
	MaxFreeParents int `json:"maxFreeParents"` // maximum number of parents with no gas cost

	MaxEpochBlocks   idx.Frame     `json:"maxEpochBlocks"`
	MaxEpochDuration time.Duration `json:"maxEpochDuration"`

	MaxValidatorEventsInBlock idx.Event `json:"maxValidatorEventsInBlock"`
}

// Config describes network net.
type Config struct {
	Name      string
	NetworkID uint64

	Genesis       genesis.Genesis
	ShortGasPower GasPowerConfig `json:"shortGasPower"`
	LongGasPower  GasPowerConfig `json:"longGasPower"`

	// Graph options
	Dag DagConfig
}

// EvmChainConfig returns ChainConfig for transaction signing and execution
func (c *Config) EvmChainConfig() *ethparams.ChainConfig {
	cfg := *ethparams.AllEthashProtocolChanges
	cfg.ChainID = new(big.Int).SetUint64(c.NetworkID)
	return &cfg
}

func FakeNetConfig(accs genesis.Validators) Config {
	return Config{
		Name:          "fake",
		NetworkID:     FakeNetworkID,
		Genesis:       genesis.FakeGenesis(accs),
		Dag:           FakeNetDagConfig(),
		ShortGasPower: FakeShortGasPowerConfig(),
		LongGasPower:  FakeLongGasPowerConfig(),
	}
}

func DefaultDagConfig() DagConfig {
	return DagConfig{
		MaxParents:                10,
		MaxFreeParents:            3,
		MaxEpochBlocks:            1000,
		MaxEpochDuration:          4 * time.Hour,
		MaxValidatorEventsInBlock: 50,
	}
}

func FakeNetDagConfig() DagConfig {
	cfg := DefaultDagConfig()
	cfg.MaxEpochBlocks = 200
	cfg.MaxEpochDuration = 10 * time.Minute
	return cfg
}

// DefaulLongGasPowerConfig is long-window config
func DefaulLongGasPowerConfig() GasPowerConfig {
	return GasPowerConfig{
		InitialAllocPerSec: 100 * params.EventGas,
		MaxAllocPerSec:     1000 * params.EventGas,
		MinAllocPerSec:     10 * params.EventGas,
		MaxAllocPeriod:     inter.Timestamp(60 * time.Minute),
		StartupAllocPeriod: inter.Timestamp(5 * time.Second),
		MinStartupGas:      params.EventGas * 20,
	}
}

// DefaultShortGasPowerConfig is short-window config
func DefaultShortGasPowerConfig() GasPowerConfig {
	// 5x faster allocation rate, 12x lower max accumulated gas power
	cfg := DefaulLongGasPowerConfig()
	cfg.InitialAllocPerSec *= 5
	cfg.MaxAllocPerSec *= 5
	cfg.MinAllocPerSec *= 5
	cfg.StartupAllocPeriod /= 5
	cfg.MaxAllocPeriod /= 5 * 12
	return cfg
}

// FakeLongGasPowerConfig is fake long-window config
func FakeLongGasPowerConfig() GasPowerConfig {
	config := DefaulLongGasPowerConfig()
	config.InitialAllocPerSec *= 1000
	config.MaxAllocPerSec *= 1000
	return config
}

// FakeShortGasPowerConfig is fake short-window config
func FakeShortGasPowerConfig() GasPowerConfig {
	config := DefaultShortGasPowerConfig()
	config.InitialAllocPerSec *= 1000
	config.MaxAllocPerSec *= 1000
	return config
}
