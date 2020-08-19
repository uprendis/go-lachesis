package emitter

import (
	"math/rand"
	"time"

	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// EmitIntervals is the configuration of emit intervals.
type EmitIntervals struct {
	Min                time.Duration
	Max                time.Duration
	SelfForkProtection time.Duration
}

// Config is the configuration of events emitter.
type Config struct {
	Validator idx.ValidatorID

	EpochTailLength idx.Frame
	EmitIntervals   EmitIntervals // event emission intervals
}

// DefaultEmitterConfig returns the default configurations for the events emitter.
func DefaultEmitterConfig() Config {
	return Config{
		EmitIntervals: EmitIntervals{
			Min:                200 * time.Millisecond,
			Max:                12 * time.Minute,
			SelfForkProtection: 30 * time.Minute, // should be at least 2x of MaxEmitInterval
		},
		EpochTailLength: 3,
	}
}

// RandomizeEmitTime and return new config
func (cfg *EmitIntervals) RandomizeEmitTime(r *rand.Rand) *EmitIntervals {
	config := *cfg
	// value = value - 0.1 * value + 0.1 * random value
	if config.Max > 10 {
		config.Max = config.Max - config.Max/10 + time.Duration(r.Int63n(int64(config.Max/10)))
	}
	// value = value + 0.1 * random value
	if config.SelfForkProtection > 10 {
		config.SelfForkProtection = config.SelfForkProtection + time.Duration(r.Int63n(int64(config.SelfForkProtection/10)))
	}
	return &config
}

// FakeEmitterConfig returns the testing configurations for the events emitter.
func FakeEmitterConfig() Config {
	cfg := DefaultEmitterConfig()
	cfg.EmitIntervals.Max = 10 * time.Second // don't wait long in fakenet
	cfg.EmitIntervals.SelfForkProtection = cfg.EmitIntervals.Max * 3 / 2
	return cfg
}
