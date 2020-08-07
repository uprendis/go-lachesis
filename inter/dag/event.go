package dag

import (
	"fmt"
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

type Event interface {
	Epoch() idx.Epoch
	Seq() idx.Event

	Frame() idx.Frame
	IsRoot() bool

	Creator() idx.StakerID

	Parents() hash.Events
	SelfParent() *hash.Event
	IsSelfParent(hash hash.Event) bool

	Lamport() idx.Lamport
	ClaimedTime() inter.Timestamp
	MedianTime() inter.Timestamp

	ID() hash.Event

	String() string
}

type MutableEvent interface {

}

// BaseEvent is the consensus message in the Lachesis consensus algorithm
// The structure isn't supposed to be used as-is:
// Doesn't contain payload, it should be extended by an app
// Doesn't contain event signature, it should be extended by an app
type BaseEvent struct {
	epoch idx.Epoch
	seq   idx.Event

	frame  idx.Frame
	isRoot bool

	creator idx.StakerID

	parents hash.Events

	lamport     idx.Lamport
	claimedTime inter.Timestamp
	medianTime  inter.Timestamp

	id hash.Event
}

// MutableBaseEvent is a mutable version of BaseEvent
type MutableBaseEvent struct {
	Epoch idx.Epoch
	Seq   idx.Event

	Frame  idx.Frame
	IsRoot bool

	Creator idx.StakerID

	Parents hash.Events

	Lamport     idx.Lamport
	ClaimedTime inter.Timestamp
	MedianTime  inter.Timestamp
}

func (e *MutableBaseEvent) Build(r_id [24]byte) *BaseEvent {
	id := hash.Event{}
	copy(id[0:4], e.Epoch.Bytes())
	copy(id[4:8], e.Lamport.Bytes())
	copy(id[8:], r_id[:])
	return &BaseEvent{
		epoch:       e.Epoch,
		seq:         e.Seq,
		frame:       e.Frame,
		isRoot:      e.IsRoot,
		creator:     e.Creator,
		parents:     e.Parents,
		lamport:     e.Lamport,
		claimedTime: e.ClaimedTime,
		medianTime:  e.MedianTime,
		id:          id,
	}
}

// fmtFrame returns frame string representation.
func fmtFrame(frame idx.Frame, isRoot bool) string {
	if isRoot {
		return fmt.Sprintf("%d:y", frame)
	}
	return fmt.Sprintf("%d:n", frame)
}

// String returns string representation.
func (e *BaseEvent) String() string {
	return fmt.Sprintf("{id=%s, p=%s, by=%d, frame=%s}", e.ID().ShortID(3), e.Parents().String(), e.Creator(), fmtFrame(e.Frame(), e.IsRoot()))
}

// SelfParent returns event's self-parent, if any
func (e *BaseEvent) SelfParent() *hash.Event {
	if e.seq <= 1 || len(e.parents) == 0 {
		return nil
	}
	return &e.parents[0]
}

// IsSelfParent is true if specified ID is event's self-parent
func (e *BaseEvent) IsSelfParent(hash hash.Event) bool {
	if e.SelfParent() == nil {
		return false
	}
	return *e.SelfParent() == hash
}
