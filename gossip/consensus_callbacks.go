package gossip

import (
	"errors"
	"github.com/Fantom-foundation/lachesis-base/eventcheck"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/Fantom-foundation/go-lachesis/inter"
)

var (
	errStopped = errors.New("service is stopped")
)

func (s *Service) GetConsensusCallbacks() lachesis.ConsensusCallbacks {
	return lachesis.ConsensusCallbacks{
		BeginBlock: func(cBlock *lachesis.Block) lachesis.BlockCallbacks {
			start := time.Now()

			confirmedEvents := inter.Events{}
			var atropos *inter.Event
			payloadSize := 0

			return lachesis.BlockCallbacks{
				ApplyEvent: func(_e dag.Event) {
					e := _e.(*inter.Event)
					if cBlock.Atropos == e.ID() {
						atropos = e
					}
					if len(e.Payload()) != 0 {
						// non-empty events only
						confirmedEvents.Add(e)
					}
					payloadSize += len(e.Payload())
				},
				EndBlock: func() (sealEpoch *pos.Validators) {
					// sort events by Lamport time
					sort.Sort(confirmedEvents)

					// new block
					bs := s.store.GetBlockState()
					var block = &inter.Block{
						Time:    atropos.CreationTime(), // non-secure, may be easily biased. That's fine only for benchopera
						Atropos: cBlock.Atropos,
						Events:  confirmedEvents.IDs(),
					}
					bs.Block++
					bs.EpochBlocks++
					s.store.SetBlockState(bs)
					s.store.SetBlock(bs.Block, block)

					log.Info("New block", "index", bs.Block, "atropos", block.Atropos, "payload", payloadSize, "t", time.Since(start))

					// in benchopera, sealing condition is straightforward, based only on blocks count or cheaters present
					if bs.EpochBlocks >= s.config.Net.Dag.MaxEpochBlocks || cBlock.Cheaters.Len() != 0 {
						// seal epoch
						// in benchopera, validators group doesn't change, so just use genesis validators (even if they became cheaters)
						return s.config.Net.Genesis.Validators.Build()
					}
					return nil
				},
			}
		},
	}
}

// processEvent extends the engine.ProcessEvent with gossip-specific actions on each event processing
func (s *Service) processEvent(e *inter.Event) error {
	// s.engineMu is locked here
	if s.stopped {
		return errStopped
	}

	if s.store.HasEvent(e.ID()) { // sanity check
		return eventcheck.ErrAlreadyConnectedEvent
	}

	oldEpoch := s.store.GetEpoch()

	s.store.SetEvent(e)
	err := s.engine.ProcessEvent(e)
	if err != nil { // TODO make it possible to write only on success
		s.store.DelEvent(e.ID())
		return err
	}

	// set validator's last event. we don't care about forks, because this index is used only for emitter
	s.store.SetLastEvent(e.Epoch(), e.Creator(), e.ID())

	// track events with no descendants, i.e. heads
	for _, parent := range e.Parents() {
		s.store.DelHead(e.Epoch(), parent)
	}
	s.store.AddHead(e.Epoch(), e.ID())

	s.packsOnNewEvent(e, e.Epoch())
	s.emitter.OnNewEvent(e)

	newEpoch := s.store.GetEpoch()

	immediately := newEpoch != oldEpoch

	return s.store.Commit(e.ID().Bytes(), immediately)
}
