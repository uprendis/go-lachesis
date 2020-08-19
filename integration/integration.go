package integration

import (
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"time"

	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"

	"github.com/Fantom-foundation/go-lachesis/benchopera"
	"github.com/Fantom-foundation/go-lachesis/gossip"
)

// NewIntegration creates gossip service for the integration test
func NewIntegration(ctx *adapters.ServiceContext, network benchopera.Config, validator idx.ValidatorID) *gossip.Service {
	gossipCfg := gossip.DefaultConfig(network)

	engine, _, gdb := MakeEngine(ctx.Config.DataDir, &gossipCfg)

	gossipCfg.Emitter.Validator = validator
	gossipCfg.Emitter.EmitIntervals.Max = 3 * time.Second
	gossipCfg.Emitter.EmitIntervals.DoublesignProtection = 0

	svc, err := gossip.NewService(ctx.NodeContext, &gossipCfg, gdb, engine)
	if err != nil {
		panic(err)
	}

	return svc
}
