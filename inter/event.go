package inter

import (
	"crypto/sha256"
	"fmt"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

type Event struct {
	dag.BaseEvent
	extEvent
}

func (e *Event) CreationTime() Timestamp { return Timestamp(e.RawTime()) }

type extEvent struct {
	version uint32 // serialization version

	gasPowerLeft GasPowerLeft
	gasPowerUsed uint64

	payload []byte
	sig     []byte
}

func (e *extEvent) Version() uint32 { return e.version }

func (e *extEvent) GasPowerLeft() GasPowerLeft { return e.gasPowerLeft }

func (e *extEvent) GasPowerUsed() uint64 { return e.gasPowerUsed }

func (e *extEvent) Payload() []byte { return e.payload }

func (e *extEvent) Sig() []byte { return e.sig }

type mutableExtEvent struct {
	extEvent
}

func (e *mutableExtEvent) SetVersion(v uint32) { e.version = v }

func (e *mutableExtEvent) SetGasPowerLeft(v GasPowerLeft) { e.gasPowerLeft = v }

func (e *mutableExtEvent) SetGasPowerUsed(v uint64) { e.gasPowerUsed = v }

func (e *mutableExtEvent) SetPayload(v []byte) { e.payload = v }

func (e *mutableExtEvent) SetSig(v []byte) { e.sig = v }

type MutableEvent struct {
	dag.MutableBaseEvent
	mutableExtEvent
}

func MutableFrom(e *Event) *MutableEvent {
	return &MutableEvent{dag.MutableBaseEvent{e.BaseEvent}, mutableExtEvent{e.extEvent}}
}

func (e *MutableEvent) calcID() (id [24]byte) {
	hasher := sha256.New()
	b, err := e.immutable().MarshalBinary()
	if err != nil {
		panic("can't encode: " + err.Error())
	}
	_, err = hasher.Write(b)
	if err != nil {
		panic("can't hash: " + err.Error())
	}
	copy(id[:], hasher.Sum(nil)[:24])
	return id
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
