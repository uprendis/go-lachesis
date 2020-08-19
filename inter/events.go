package inter

import (
	"bytes"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"strings"
)

// Events is a ordered slice of events.
type Events []*Event

// String returns human readable representation.
func (ee Events) String() string {
	ss := make([]string, len(ee))
	for i := 0; i < len(ee); i++ {
		ss[i] = ee[i].String()
	}
	return strings.Join(ss, " ")
}

// Add appends hash to the slice.
func (ee *Events) Add(e ...*Event) {
	*ee = append(*ee, e...)
}

func (ee Events) IDs() hash.Events {
	res := make(hash.Events, 0, len(ee))
	for _, e := range ee {
		res.Add(e.ID())
	}
	return res
}

func (hh Events) Len() int      { return len(hh) }
func (hh Events) Swap(i, j int) { hh[i], hh[j] = hh[j], hh[i] }
func (hh Events) Less(i, j int) bool {
	return bytes.Compare(hh[i].ID().Bytes(), hh[j].ID().Bytes()) < 0
}
