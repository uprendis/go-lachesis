package poset

import (
	"github.com/Fantom-foundation/go-lachesis/inter/dag"
	"github.com/Fantom-foundation/go-lachesis/inter/dag/tdag"
	"github.com/Fantom-foundation/go-lachesis/lachesis"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/Fantom-foundation/go-lachesis/logger"
)

func TestConfirmBlockEvents(t *testing.T) {
	logger.SetTestMode(t)
	assertar := assert.New(t)

	nodes := tdag.GenNodes(5)
	poset, _, input := FakePoset("", nodes)

	var (
		frames []idx.Frame
		blocks []*lachesis.Block
	)
	applyBlock := poset.callback.ApplyBlock
	poset.callback.ApplyBlock = func(block *lachesis.Block, decidedFrame idx.Frame, cheaters lachesis.Cheaters) (hash.Hash, bool) {
		frames = append(frames, poset.LastDecidedFrame)
		blocks = append(blocks, block)

		return applyBlock(block, decidedFrame, cheaters)
	}

	eventCount := int(poset.dag.MaxEpochBlocks)
	_ = tdag.ForEachRandEvent(nodes, eventCount, 5, nil, tdag.ForEachEvent{
		Process: func(e *dag.Event, name string) {
			input.SetEvent(e)
			assertar.NoError(
				poset.ProcessEvent(e))
			assertar.NoError(
				flushDb(poset, e.ID()))

		},
		Build: func(e *dag.Event, name string) *dag.Event {
			e.Epoch = idx.Epoch(1)
			if e.Seq%2 != 0 {
				e.Transactions = append(e.Transactions, &types.Transaction{})
			}
			e.TxHash = types.DeriveSha(e.Transactions)
			return poset.Prepare(e)
		},
	})

	// unconfirm all events
	it := poset.store.table.ConfirmedEvent.NewIterator()
	batch := poset.store.table.ConfirmedEvent.NewBatch()
	for it.Next() {
		assertar.NoError(batch.Delete(it.Key()))
	}
	assertar.NoError(batch.Write())
	it.Release()

	for i, block := range blocks {
		frame := frames[i]
		atropos := blocks[i].Atropos

		// call confirmBlock again
		gotBlock, cheaters := poset.confirmBlock(frame, atropos)

		if !assertar.Empty(cheaters) {
			break
		}
		if !assertar.Equal(block.Events, gotBlock.Events) {
			break
		}
	}
}
