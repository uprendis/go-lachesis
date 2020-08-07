package tdag

import (
	"fmt"
	"github.com/Fantom-foundation/go-lachesis/inter/dag"
	"golang.org/x/crypto/sha3"
	"math/rand"

	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

// GenNodes generates nodes.
// Result:
//   - nodes  is an array of node addresses;
func GenNodes(
	nodeCount int,
) (
	nodes []idx.StakerID,
) {
	// init results
	nodes = make([]idx.StakerID, nodeCount)
	// make and name nodes
	for i := 0; i < nodeCount; i++ {
		addr := hash.FakePeer()
		nodes[i] = addr
		hash.SetNodeName(addr, "node"+string('A'+i))
	}

	return
}

// ForEachRandFork generates random events with forks for test purpose.
// Result:
//   - callbacks are called for each new event;
//   - events maps node address to array of its events;
func ForEachRandFork(
	nodes []idx.StakerID,
	cheatersArr []idx.StakerID,
	eventCount int,
	parentCount int,
	forksCount int,
	r *rand.Rand,
	callback ForEachEvent,
) (
	events map[idx.StakerID][]dag.Event,
) {
	if r == nil {
		// fixed seed
		r = rand.New(rand.NewSource(0))
	}
	// init results
	nodeCount := len(nodes)
	events = make(map[idx.StakerID][]dag.Event, nodeCount)
	cheaters := map[idx.StakerID]int{}
	for _, cheater := range cheatersArr {
		cheaters[cheater] = 0
	}

	// make events
	for i := 0; i < nodeCount*eventCount; i++ {
		// seq parent
		self := i % nodeCount
		creator := nodes[self]
		parents := r.Perm(nodeCount)
		for j, n := range parents {
			if n == self {
				parents = append(parents[0:j], parents[j+1:]...)
				break
			}
		}
		parents = parents[:parentCount-1]
		// make
		te := TestEvent{}
		te.SetCreator(creator)
		te.SetParents(hash.Events{})
		// first parent is a last creator's event or empty hash
		var parent dag.Event
		if ee := events[creator]; len(ee) > 0 {
			parent = ee[len(ee)-1]

			// may insert fork
			forksAlready, isCheater := cheaters[creator]
			forkPossible := len(ee) > 1
			forkLimitOk := forksAlready < forksCount
			forkFlipped := r.Intn(eventCount) <= forksCount || i < (nodeCount-1)*eventCount
			if isCheater && forkPossible && forkLimitOk && forkFlipped {
				parent = ee[r.Intn(len(ee)-1)]
				if r.Intn(len(ee)) == 0 {
					parent = nil
				}
				cheaters[creator]++
			}
		}
		if parent == nil {
			te.SetSeq(1)
			te.SetLamport(1)
		} else {
			te.SetSeq(parent.Seq() + 1)
			te.AddParent(parent.ID())
			te.SetLamport(parent.Lamport() + 1)
		}
		// other parents are the lasts other's events
		for _, other := range parents {
			if ee := events[nodes[other]]; len(ee) > 0 {
				parent := ee[len(ee)-1]
				te.AddParent(parent.ID())
				if te.Lamport() <= parent.Lamport() {
					te.SetLamport(parent.Lamport() + 1)
				}
			}
		}
		name := fmt.Sprintf("%s%03d", string('a'+self), len(events[creator]))
		// buildEvent callback
		var e dag.Event
		if callback.Build != nil {
			e = callback.Build(&te, name)
		}
		if e == nil {
			continue
		}
		// save and name event
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(EventToBytes(e))
		id := [24]byte{}
		copy(id[:], hasher.Sum(nil)[:24])
		e.SetID(id)
		hash.SetEventName(e.ID(), fmt.Sprintf("%s%03d", string('a'+self), len(events[creator])))
		events[creator] = append(events[creator], e)
		// callback
		if callback.Process != nil {
			callback.Process(e, name)
		}
	}

	return
}

// ForEachRandEvent generates random events for test purpose.
// Result:
//   - callbacks are called for each new event;
//   - events maps node address to array of its events;
func ForEachRandEvent(
	nodes []idx.StakerID,
	eventCount int,
	parentCount int,
	r *rand.Rand,
	callback ForEachEvent,
) (
	events map[idx.StakerID][]*dag.BaseEvent,
) {
	return ForEachRandFork(nodes, []idx.StakerID{}, eventCount, parentCount, 0, r, callback)
}

// GenRandEvents generates random events for test purpose.
// Result:
//   - events maps node address to array of its events;
func GenRandEvents(
	nodes []idx.StakerID,
	eventCount int,
	parentCount int,
	r *rand.Rand,
) (
	events map[idx.StakerID][]*dag.BaseEvent,
) {
	return ForEachRandEvent(nodes, eventCount, parentCount, r, ForEachEvent{})
}

func delPeerIndex(events map[idx.StakerID][]*dag.Event) (res dag.Events) {
	for _, ee := range events {
		res = append(res, ee...)
	}
	return
}
