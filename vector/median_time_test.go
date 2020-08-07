package vector

import (
	dag2 "github.com/Fantom-foundation/go-lachesis/inter/dag"
	"github.com/Fantom-foundation/go-lachesis/inter/dag/tdag"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/pos"
	"github.com/Fantom-foundation/go-lachesis/kvdb/memorydb"
	"github.com/Fantom-foundation/go-lachesis/logger"
)

func TestMedianTimeOnIndex(t *testing.T) {
	logger.SetTestMode(t)

	nodes := tdag.GenNodes(5)
	weights := []pos.Stake{5, 4, 3, 2, 1}
	validators := pos.ArrayToValidators(nodes, weights)

	vi := NewIndex(DefaultIndexConfig(), validators, memorydb.New(), nil)

	assertar := assert.New(t)
	{ // seq=0
		e := inter.NewEvent().Event.ID()
		// validator indexes are sorted by stake amount
		beforeSeq := NewHighestBeforeSeq(validators.Len())
		beforeTime := NewHighestBeforeTime(validators.Len())

		beforeSeq.Set(0, BranchSeq{Seq: 0})
		beforeTime.Set(0, 100)

		beforeSeq.Set(1, BranchSeq{Seq: 0})
		beforeTime.Set(1, 100)

		beforeSeq.Set(2, BranchSeq{Seq: 1})
		beforeTime.Set(2, 10)

		beforeSeq.Set(3, BranchSeq{Seq: 1})
		beforeTime.Set(3, 10)

		beforeSeq.Set(4, BranchSeq{Seq: 1})
		beforeTime.Set(4, 10)

		vi.SetHighestBefore(e, beforeSeq, beforeTime)
		assertar.Equal(inter.Timestamp(1), vi.MedianTime(e, 1))
	}

	{ // fork seen = true
		e := inter.NewEvent().Event.ID()
		// validator indexes are sorted by stake amount
		beforeSeq := NewHighestBeforeSeq(validators.Len())
		beforeTime := NewHighestBeforeTime(validators.Len())

		beforeSeq.Set(0, forkDetectedSeq)
		beforeTime.Set(0, 100)

		beforeSeq.Set(1, forkDetectedSeq)
		beforeTime.Set(1, 100)

		beforeSeq.Set(2, BranchSeq{Seq: 1})
		beforeTime.Set(2, 10)

		beforeSeq.Set(3, BranchSeq{Seq: 1})
		beforeTime.Set(3, 10)

		beforeSeq.Set(4, BranchSeq{Seq: 1})
		beforeTime.Set(4, 10)

		vi.SetHighestBefore(e, beforeSeq, beforeTime)
		assertar.Equal(inter.Timestamp(10), vi.MedianTime(e, 1))
	}

	{ // normal
		e := inter.NewEvent().Event.ID()
		// validator indexes are sorted by stake amount
		beforeSeq := NewHighestBeforeSeq(validators.Len())
		beforeTime := NewHighestBeforeTime(validators.Len())

		beforeSeq.Set(0, BranchSeq{Seq: 1})
		beforeTime.Set(0, 11)

		beforeSeq.Set(1, BranchSeq{Seq: 2})
		beforeTime.Set(1, 12)

		beforeSeq.Set(2, BranchSeq{Seq: 2})
		beforeTime.Set(2, 13)

		beforeSeq.Set(3, BranchSeq{Seq: 3})
		beforeTime.Set(3, 14)

		beforeSeq.Set(4, BranchSeq{Seq: 4})
		beforeTime.Set(4, 15)

		vi.SetHighestBefore(e, beforeSeq, beforeTime)
		assertar.Equal(inter.Timestamp(12), vi.MedianTime(e, 1))
	}

}

func TestMedianTimeOnDAG(t *testing.T) {
	logger.SetTestMode(t)

	dag := `
 ║
 nodeA001
 ║
 nodeA012
 ║            ║
 ║            nodeB001
 ║            ║            ║
 ║            ╠═══════════ nodeC001
 ║║           ║            ║            ║
 ║╚══════════─╫─══════════─╫─══════════ nodeD001
║║            ║            ║            ║
╚ nodeA002════╬════════════╬════════════╣
 ║║           ║            ║            ║
 ║╚══════════─╫─══════════─╫─══════════ nodeD002
 ║            ║            ║            ║
 nodeA003════─╫─══════════─╫─═══════════╣
 ║            ║            ║
 ╠════════════nodeB002     ║
 ║            ║            ║
 ╠════════════╫═══════════ nodeC002
`

	weights := []pos.Stake{3, 4, 2, 1}
	genesisTime := inter.Timestamp(1)
	claimedTimes := map[string]inter.Timestamp{
		"nodeA001": inter.Timestamp(111),
		"nodeB001": inter.Timestamp(112),
		"nodeC001": inter.Timestamp(13),
		"nodeD001": inter.Timestamp(14),
		"nodeA002": inter.Timestamp(120),
		"nodeD002": inter.Timestamp(20),
		"nodeA012": inter.Timestamp(120),
		"nodeA003": inter.Timestamp(20),
		"nodeB002": inter.Timestamp(20),
		"nodeC002": inter.Timestamp(35),
	}
	medianTimes := map[string]inter.Timestamp{
		"nodeA001": genesisTime,
		"nodeB001": genesisTime,
		"nodeC001": inter.Timestamp(13),
		"nodeD001": genesisTime,
		"nodeA002": inter.Timestamp(112),
		"nodeD002": genesisTime,
		"nodeA012": genesisTime,
		"nodeA003": inter.Timestamp(20),
		"nodeB002": inter.Timestamp(20),
		"nodeC002": inter.Timestamp(35),
	}
	t.Run("testMedianTimeOnDAG", func(t *testing.T) {
		testMedianTime(t, dag, weights, claimedTimes, medianTimes, genesisTime)
	})
}

func testMedianTime(t *testing.T, dag string, weights []pos.Stake, claimedTimes map[string]inter.Timestamp, medianTimes map[string]inter.Timestamp, genesis inter.Timestamp) {
	assertar := assert.New(t)

	var ordered []*dag2.Event
	nodes, _, named := tdag.ASCIIschemeForEach(dag, tdag.ForEachEvent{
		Build: func(e *dag2.Event, name string) *dag2.Event {
			e.ClaimedTime = claimedTimes[name]
			return e
		},
		Process: func(e *dag2.Event, name string) {
			ordered = append(ordered, e)
		},
	})

	validators := pos.ArrayToValidators(nodes, weights)

	events := make(map[hash.Event]*dag2.Event)
	getEvent := func(id hash.Event) *dag2.Event {
		return events[id]
	}

	vi := NewIndex(DefaultIndexConfig(), validators, memorydb.New(), getEvent)

	// push
	for _, e := range ordered {
		events[e.ID()] = &e.Event
		vi.Add(&e.Event)
		vi.Flush()
	}

	// check
	for name, e := range named {
		expected, ok := medianTimes[name]
		if !ok {
			continue
		}
		assertar.Equal(expected, vi.MedianTime(e.ID(), genesis), name)
	}
}
