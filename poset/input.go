package poset

import (
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter/dag"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

// EventSource is a callback for getting events from an external storage.
type EventSource interface {
	HasEvent(hash.Event) bool
	GetEvent(hash.Event) *dag.Event
	GetEvent(idx.Epoch, hash.Event) *dag.Event
}

/*
 * Poset's methods:
 */

// GetEvent returns event.
func (p *Poset) GetEvent(h hash.Event) *Event {
	e := p.input.GetEvent(h)
	if e == nil {
		p.Log.Crit("Got unsaved event", "event", h.String())
	}
	return &Event{
		Event: e,
	}
}

// GetEvent returns event header.
func (p *Poset) GetEvent(epoch idx.Epoch, h hash.Event) *dag.Event {
	return p.input.GetEvent(epoch, h)
}
