package poset

import (
	"bytes"
	"sort"

	"github.com/Fantom-foundation/go-lachesis/src/hash"
	"github.com/Fantom-foundation/go-lachesis/src/inter"
	"github.com/Fantom-foundation/go-lachesis/src/inter/idx"
)

func (p *Poset) fareOrdering(frame idx.Frame, atropos hash.Event, unordered []*inter.EventHeaderData) hash.Events {
	// sort by lamport timestamp & hash
	sort.Slice(unordered, func(i, j int) bool {
		a, b := unordered[i], unordered[j]

		if a.Lamport != b.Lamport {
			return a.Lamport < b.Lamport
		}

		return bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes()) < 0
	})
	ordered := unordered

	// calculate difference between highest and lowest period
	highestLamport := ordered[len(ordered)-1].Lamport
	lowestLamport := ordered[0].Lamport
	frameLamportPeriod := idx.MaxLamport(highestLamport-lowestLamport+1, 1)

	// calculate difference between atropos's median time and previous atropos's consensus time (almost the same as previous median time)
	nowMedianTime := p.GetEventHeader(p.EpochN, atropos).MedianTime
	frameTimePeriod := inter.MaxTimestamp(nowMedianTime-p.LastConsensusTime, 1)
	if p.LastConsensusTime > nowMedianTime {
		frameTimePeriod = 1
	}

	// Calculate time ratio & time offset
	timeRatio := inter.MaxTimestamp(frameTimePeriod/inter.Timestamp(frameLamportPeriod), 1)

	lowestConsensusTime := p.LastConsensusTime + timeRatio
	timeOffset := int64(lowestConsensusTime) - int64(lowestLamport)*int64(timeRatio)

	// Calculate consensus timestamp of an event with highestLamport (it's always atropos)
	p.LastConsensusTime = inter.Timestamp(int64(highestLamport)*int64(timeRatio) + timeOffset)

	// Save new timeRatio & timeOffset to frame
	p.store.SetFrameInfo(p.EpochN, frame, &FrameInfo{
		TimeOffset: timeOffset,
		TimeRatio:  timeRatio,
	})

	ids := make(hash.Events, len(ordered))
	for i, e := range ordered {
		ids[i] = e.Hash()
	}
	return ids
}