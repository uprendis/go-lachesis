package tdag

import (
	"github.com/Fantom-foundation/go-lachesis/inter/dag"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
)

// TODO eliminate dependency on RLP in tests

// EventToBytes serializes events
func EventToBytes(e *dag.Event) []byte {
	b, _ := rlp.EncodeToBytes(e)
	return b
}

// BytesToEvent deserializes event from bytes
func BytesToEvent(b []byte) (*dag.Event, error) {
	e := &dag.Event{}
	err := rlp.DecodeBytes(b, e)
	return e, err
}

// DecodeEvent deserializes event
func DecodeEvent(r io.Reader) (*dag.Event, error) {
	e := &dag.Event{}
	err := rlp.Decode(r, e)
	return e, err
}
