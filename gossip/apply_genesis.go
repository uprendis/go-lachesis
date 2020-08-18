package gossip

import (
	"fmt"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/Fantom-foundation/go-lachesis/network"
)

// GenesisMismatchError is raised when trying to overwrite an existing
// genesis block with an incompatible one.
type GenesisMismatchError struct {
	Stored, New hash.Event
}

// Error implements error interface.
func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database contains incompatible gossip genesis (have %s, new %s)", e.Stored.FullID(), e.New.FullID())
}

// ApplyGenesis writes initial state.
func (s *Store) ApplyGenesis(net *network.Config) (genesisAtropos hash.Event, genesisState common.Hash, new bool, err error) {
	storedGenesis := s.GetBlock(0)
	if storedGenesis != nil {
		newHash := calcGenesisHash(net)
		if storedGenesis.Atropos != newHash {
			return genesisAtropos, genesisState, true, &GenesisMismatchError{storedGenesis.Atropos, newHash}
		}

		genesisAtropos = storedGenesis.Atropos
		genesisState = common.Hash(genesisAtropos)
		return genesisAtropos, genesisState, false, nil
	}
	// if we'here, then it's first time genesis is applied
	genesisAtropos, genesisState, err = s.applyGenesis(net)
	if err != nil {
		return genesisAtropos, genesisState, true, err
	}

	return genesisAtropos, genesisState, true, err
}

// calcGenesisHash calcs hash of genesis state.
func calcGenesisHash(net *network.Config) hash.Event {
	s := NewMemStore()
	defer s.Close()

	h, _, _ := s.applyGenesis(net)

	return h
}

func (s *Store) applyGenesis(net *network.Config) (genesisAtropos hash.Event, genesisState common.Hash, err error) {
	// apply app genesis
	state, err := s.app.ApplyGenesis(net)
	if err != nil {
		return genesisAtropos, genesisState, err
	}

	prettyHash := func(net *network.Config) hash.Event {
		e := inter.NewEvent()
		// for nice-looking ID
		e.Epoch = 0
		e.Lamport = idx.Lamport(net.Dag.MaxEpochBlocks)
		// actual data hashed
		e.Extra = net.Genesis.ExtraData
		e.ClaimedTime = net.Genesis.Time
		e.TxHash = net.Genesis.Alloc.Accounts.Hash()

		return e.CalcHash()
	}
	genesisAtropos = prettyHash(net)
	genesisState = common.Hash(genesisAtropos)

	block := &inter.Block{
		Time:    net.Genesis.Time,
		Atropos: genesisAtropos,
		Events:  hash.Events{genesisAtropos},
	}
	s.SetBlock(0, block)

	return genesisAtropos, genesisState, nil
}
