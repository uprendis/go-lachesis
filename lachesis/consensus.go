package lachesis

import (
	"github.com/Fantom-foundation/go-lachesis/inter/dag"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/Fantom-foundation/go-lachesis/inter/pos"
	"github.com/Fantom-foundation/go-lachesis/vector"
)

// Consensus is a consensus interface.
type Consensus interface {
	// PushEvent takes event for processing.
	ProcessEvent(e *dag.Event) error
	// GetVectorIndex returns internal vector clock if exists
	GetVectorIndex() *vector.Index
	// Fill sets consensus fields. Returns nil if event should be dropped.
	Fill(e *dag.Event) *dag.Event

	// Bootstrap must be called (once) before calling other methods
	Bootstrap(callbacks ConsensusCallbacks)
}

// ConsensusCallbacks contains callbacks called during block processing by consensus engine
type ConsensusCallbacks struct {
	// ApplyBlock is callback type to apply the new block to the state
	ApplyBlock func(block *Block) (sealEpoch bool, newValidators *pos.Validators)
	// OnEventConfirmed is callback type to notify about event confirmation.
	OnEventConfirmed func(event *dag.Event, seqDepth idx.Event)
	// IsEventAllowedIntoBlock is callback type to check is event may be within block or not
	IsEventAllowedIntoBlock func(event *dag.Event, seqDepth idx.Event) bool
}
