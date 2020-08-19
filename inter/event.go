package inter

import (
	"crypto/sha256"
	"fmt"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

const SigSize = 64

type Event struct {
	dag.BaseEvent
	extEvent
}

func (e *Event) CreationTime() Timestamp { return Timestamp(e.RawTime()) }

func (e *Event) HashToSign() hash.Hash {
	hasher := sha256.New()
	b, err := e.MarshalBinary()
	if err != nil {
		panic("can't encode: " + err.Error())
	}
	_, err = hasher.Write(b[:len(b) - SigSize]) // don't hash the signature
	if err != nil {
		panic("can't hash: " + err.Error())
	}
	return hash.BytesToHash(hasher.Sum(nil))
}

type extEvent struct {
	version uint8 // serialization version

	payload []byte
	sig     [SigSize]byte
}

func (e *extEvent) Version() uint8 { return e.version }

func (e *extEvent) Payload() []byte { return e.payload }

func (e *extEvent) Sig() [SigSize]byte { return e.sig }

type mutableExtEvent struct {
	extEvent
}

func (e *mutableExtEvent) SetVersion(v uint8) { e.version = v }

func (e *mutableExtEvent) SetPayload(v []byte) { e.payload = v }

func (e *mutableExtEvent) SetSig(v [SigSize]byte) { e.sig = v }

type MutableEvent struct {
	dag.MutableBaseEvent
	mutableExtEvent
}

func MutableFrom(e *Event) *MutableEvent {
	return &MutableEvent{dag.MutableBaseEvent{e.BaseEvent}, mutableExtEvent{e.extEvent}}
}

func (e *MutableEvent) calcID() (id [24]byte) {
	h := e.HashToSign()
	copy(id[:], h[:24])
	return id
}

func (e *MutableEvent) HashToSign() hash.Hash {
	return e.immutable().HashToSign()
}

func (e *MutableEvent) immutable() *Event {
	return &Event{e.MutableBaseEvent.BaseEvent, e.extEvent}
}

func (e *MutableEvent) Build() *Event {
	c := *e
	c.SetID(e.calcID())
	return c.immutable()
}

// fmtFrame returns frame string representation.
func FmtFrame(frame idx.Frame, isRoot bool) string {
	if isRoot {
		return fmt.Sprintf("%d:y", frame)
	}
	return fmt.Sprintf("%d:n", frame)
}
