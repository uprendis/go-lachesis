package lachesis

import (
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

// Block is a "chain" block.
type Block struct {
	Index    idx.Block
	Time     inter.Timestamp
	Atropos  hash.Event
	Events   hash.Events
	Cheaters Cheaters
}

// NewBlock makes block from topological ordered events.
func NewBlock(index idx.Block, time inter.Timestamp, atropos hash.Event, events hash.Events, cheaters Cheaters) *Block {
	return &Block{
		Index:    index,
		Time:     time,
		Events:   events,
		Atropos:  atropos,
		Cheaters: cheaters,
	}
}
