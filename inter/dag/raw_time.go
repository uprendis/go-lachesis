package dag

import (
	"github.com/Fantom-foundation/go-lachesis/common/bigendian"
)

type (
	// RawTimestamp is a logical rawTime. The unit is specified.
	RawTimestamp uint64
)

// Bytes gets the byte representation of the index.
func (t RawTimestamp) Bytes() []byte {
	return bigendian.Uint64ToBytes(uint64(t))
}

// BytesToRawTimestamp converts bytes to timestamp.
func BytesToRawTimestamp(b []byte) RawTimestamp {
	return RawTimestamp(bigendian.BytesToUint64(b))
}
