package emitter

import (
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// EventSource is a callback for getting events from an external storage.
type EventSource interface {
	GetEvent(hash.Event) *inter.Event
	GetLastEvent(from idx.ValidatorID) *hash.Event
	GetHeads() hash.Events
}
