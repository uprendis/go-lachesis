package poset

import (
	"github.com/Fantom-foundation/go-lachesis/inter/dag"
	"strings"
)

/*
 * Event:
 */

// Event is a poset event for internal purpose.
type Event struct {
	*dag.Event
}

/*
 * Events:
 */

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

// UnWrap extracts inter.Event.
func (ee Events) UnWrap() dag.Events {
	res := make(dag.Events, len(ee))
	for i, e := range ee {
		res[i] = e.Event
	}

	return res
}
