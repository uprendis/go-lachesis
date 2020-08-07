package tdag

import (
	"github.com/ethereum/go-ethereum/rlp"
	"io"
)

// TODO eliminate dependency on RLP in tests

// EventToBytes serializes events
func EventToBytes(e *TestEvent) []byte {
	b, _ := rlp.EncodeToBytes(e)
	return b
}

// BytesToEvent deserializes event from bytes
func BytesToEvent(b []byte) (*TestEvent, error) {
	e := &TestEvent{}
	err := rlp.DecodeBytes(b, e)
	return e, err
}

// DecodeEvent deserializes event
func DecodeEvent(r io.Reader) (*TestEvent, error) {
	e := &TestEvent{}
	err := rlp.Decode(r, e)
	return e, err
}
