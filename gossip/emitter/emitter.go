package emitter

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Fantom-foundation/go-lachesis/benchopera/genesis"
	"github.com/Fantom-foundation/lachesis-base/emitter/ancestor"
	"github.com/Fantom-foundation/lachesis-base/emitter/doublesign"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/inter/pos"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/Fantom-foundation/go-lachesis/eventcheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/basiccheck"
	"github.com/Fantom-foundation/go-lachesis/gossip/piecefunc"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/Fantom-foundation/go-lachesis/inter"

	"github.com/Fantom-foundation/go-lachesis/benchopera"
	"github.com/Fantom-foundation/go-lachesis/logger"
	"github.com/Fantom-foundation/go-lachesis/utils/errlock"
)

// EmitterWorld is emitter's external world
type EmitterWorld struct {
	Store              EventSource
	EngineMu           *sync.RWMutex
	GetEpochValidators func() (idx.Epoch, *pos.Validators)
	Epoch              func() idx.Epoch
	ProcessEvent       func(*inter.Event) error
	DagIndex           ancestor.DagIndex

	Checkers *eventcheck.Checkers

	IsSynced func() bool
	PeersNum func() int
}

type Emitter struct {
	net    *benchopera.Config
	config *Config

	world   EmitterWorld
	privKey *ecdsa.PrivateKey

	syncStatus selfForkProtection

	prevEmittedTime time.Time

	intervals EmitIntervals

	done chan struct{}
	wg   sync.WaitGroup

	logger.Periodic
}

type selfForkProtection struct {
	startup                       time.Time
	lastConnected                 time.Time
	p2pSynced                     time.Time
	prevLocalEmittedID            hash.Event
	lastExternalSelfEvent         time.Time
	lastExternalSelfEventDetected time.Time
	becameValidator               time.Time
}

// NewEmitter creation.
func NewEmitter(
	net *benchopera.Config,
	privKey *ecdsa.PrivateKey,
	config *Config,
	world EmitterWorld,
) *Emitter {

	loggerInstance := logger.MakeInstance()
	return &Emitter{
		net:       net,
		config:    config,
		privKey:   privKey,
		world:     world,
		intervals: config.EmitIntervals,
		Periodic:  logger.Periodic{Instance: loggerInstance},
	}
}

// init emitter without starting events emission
func (em *Emitter) init() {
	em.syncStatus.lastConnected = time.Now()
	em.syncStatus.startup = time.Now()
	epoch, validators := em.world.GetEpochValidators()
	em.OnNewEpoch(validators, epoch)
}

// StartEventEmission starts event emission.
func (em *Emitter) StartEventEmission() {
	if em.done != nil {
		return
	}
	em.done = make(chan struct{})

	em.init()

	done := em.done
	em.wg.Add(1)
	go func() {
		defer em.wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				// track synced time
				if em.world.PeersNum() == 0 {
					em.syncStatus.lastConnected = time.Now() // connected time ~= last time when it's true that "not connected yet"
				}
				if !em.world.IsSynced() {
					em.syncStatus.p2pSynced = time.Now() // synced time ~= last time when it's true that "not synced yet"
				}

				// must pass at least MinEmitInterval since last event
				if time.Since(em.prevEmittedTime) >= em.intervals.Min {
					em.EmitEvent()
				}
			case <-done:
				return
			}
		}
	}()
}

// StopEventEmission stops event emission.
func (em *Emitter) StopEventEmission() {
	if em.done == nil {
		return
	}

	close(em.done)
	em.done = nil
	em.wg.Wait()
}

func (em *Emitter) loadPrevEmitTime() time.Time {
	if em.config.Validator == 0 {
		return em.prevEmittedTime
	}

	prevEventID := em.world.Store.GetLastEvent(em.config.Validator)
	if prevEventID == nil {
		return em.prevEmittedTime
	}
	prevEvent := em.world.Store.GetEvent(*prevEventID)
	if prevEvent == nil {
		return em.prevEmittedTime
	}
	return prevEvent.CreationTime().Time()
}

func (em *Emitter) findBestParents(myValidatorID idx.ValidatorID) (*hash.Event, hash.Events, bool) {
	selfParent := em.world.Store.GetLastEvent(myValidatorID)
	heads := em.world.Store.GetHeads() // events with no descendants

	var strategy ancestor.SearchStrategy
	dagIndex := em.world.DagIndex
	if dagIndex != nil {
		_, validators := em.world.GetEpochValidators()
		strategy = ancestor.NewCasualityStrategy(dagIndex, validators)
		if rand.Intn(20) == 0 { // every 20th event uses random strategy is avoid repeating patterns in DAG
			strategy = ancestor.NewRandomStrategy(rand.New(rand.NewSource(time.Now().UnixNano())))
		}
	} else {
		// use dummy strategy in tests
		strategy = ancestor.NewRandomStrategy(nil)
	}

	_, parents := ancestor.FindBestParents(em.net.Dag.MaxParents, heads, selfParent, strategy)
	return selfParent, parents, true
}

// createEvent is not safe for concurrent use.
func (em *Emitter) createEvent(poolTxs map[common.Address]types.Transactions) *inter.Event {
	if em.config.Validator == 0 {
		// not a validator
		return nil
	}

	if synced := em.logSyncStatus(em.isSynced()); !synced {
		// I'm reindexing my old events, so don't create events until connect all the existing self-events
		return nil
	}

	var (
		selfParentSeq  idx.Event
		selfParentTime inter.Timestamp
		parents        hash.Events
		maxLamport     idx.Lamport
	)

	// Find parents
	selfParent, parents, ok := em.findBestParents(epoch, em.config.Validator)
	if !ok {
		return nil
	}

	// Set parent-dependent fields
	parentHeaders := make([]*inter.Event, len(parents))
	for i, p := range parents {
		parent := em.world.Store.GetEvent(epoch, p)
		if parent == nil {
			em.Log.Crit("Emitter: head not found", "event", p.String())
		}
		parentHeaders[i] = parent
		if parentHeaders[i].Creator == em.config.Validator && i != 0 {
			// there're 2 heads from me, i.e. due to a fork, findBestParents could have found multiple self-parents
			em.Periodic.Error(5*time.Second, "I've created a fork, events emitting isn't allowed", "creator", em.config.Validator)
			return nil
		}
		maxLamport = idx.MaxLamport(maxLamport, parent.Lamport)
	}

	selfParentSeq = 0
	selfParentTime = 0
	var selfParentHeader *inter.Event
	if selfParent != nil {
		selfParentHeader = parentHeaders[0]
		selfParentSeq = selfParentHeader.Seq
		selfParentTime = selfParentHeader.ClaimedTime
	}

	event := inter.NewEvent()
	event.Epoch = epoch
	event.Seq = selfParentSeq + 1
	event.Creator = em.config.Validator

	event.Parents = parents
	event.Lamport = maxLamport + 1
	event.ClaimedTime = inter.MaxTimestamp(inter.Timestamp(time.Now().UnixNano()), selfParentTime+1)

	// add version
	if em.world.AddVersion != nil {
		event = em.world.AddVersion(event)
	}

	// set consensus fields
	event = em.world.Engine.Prepare(event)
	if event == nil {
		em.Log.Warn("Dropped event while emitting")
		return nil
	}

	// calc initial GasPower
	validators := em.world.Engine.GetValidators()
	event.GasPowerUsed = basiccheck.CalcGasPowerUsed(event, &em.net.Dag)
	availableGasPower, err := em.world.Checkers.Gaspowercheck.CalcGasPower(&event.Event, selfParentHeader)
	if err != nil {
		em.Log.Warn("Gas power calculation failed", "err", err)
		return nil
	}
	if event.GasPowerUsed > availableGasPower.Min() {
		em.Periodic.Warn(time.Second, "Not enough gas power to emit event. Too small stake?",
			"gasPower", availableGasPower,
			"stake%", 100*float64(validators.Get(em.config.Validator))/float64(validators.TotalStake()))
		return nil
	}
	event.GasPowerLeft = *availableGasPower.Sub(event.GasPowerUsed)

	// Add txs
	event = em.addTxs(event, poolTxs)

	if !em.isAllowedToEmit(event, selfParentHeader) {
		return nil
	}

	// calc Merkle root
	event.TxHash = types.DeriveSha(event.Transactions)

	// sign
	myAddress := em.myAddress
	signer := func(data []byte) (sig []byte, err error) {
		acc := accounts.Account{
			Address: myAddress,
		}
		w, err := em.world.Am.Find(acc)
		if err != nil {
			return
		}
		return w.SignData(acc, MimetypeEvent, data)
	}
	if err := event.Sign(signer); err != nil {
		em.Periodic.Error(time.Second, "Failed to sign event. Please unlock account.", "err", err)
		return nil
	}
	// calc hash after event is fully built
	event.RecacheHash()
	event.RecacheSize()
	{
		// sanity check
		if em.world.Checkers != nil {
			if err := em.world.Checkers.Validate(event, parentHeaders); err != nil {
				em.Periodic.Error(time.Second, "Signed event incorrectly", "err", err)
				return nil
			}
		}
	}

	// set event name for debug
	em.nameEventForDebug(event)

	return event
}

var (
	confirmingEmitIntervalPieces = []piecefunc.Dot{
		{
			X: 0,
			Y: 1.0 * piecefunc.PercentUnit,
		},
		{
			X: 0.33 * piecefunc.PercentUnit,
			Y: 1.05 * piecefunc.PercentUnit,
		},
		{
			X: 0.66 * piecefunc.PercentUnit,
			Y: 1.20 * piecefunc.PercentUnit,
		},
		{
			X: 0.8 * piecefunc.PercentUnit,
			Y: 1.5 * piecefunc.PercentUnit,
		},
		{
			X: 0.9 * piecefunc.PercentUnit,
			Y: 3 * piecefunc.PercentUnit,
		},
		{
			X: 1.0 * piecefunc.PercentUnit,
			Y: 3.9 * piecefunc.PercentUnit,
		},
	}
	maxEmitIntervalPieces = []piecefunc.Dot{
		{
			X: 0,
			Y: 1.0 * piecefunc.PercentUnit,
		},
		{
			X: 1.0 * piecefunc.PercentUnit,
			Y: 0.89 * piecefunc.PercentUnit,
		},
	}
)

// OnNewEpoch should be called after each epoch change, and on startup
func (em *Emitter) OnNewEpoch(newValidators *genesis.Validators, newEpoch idx.Epoch) {
	// update myValidatorID
	em.config.Validator, _ = em.findMyValidatorID()
	em.prevEmittedTime = em.loadPrevEmitTime()

	// validators with lower stake should emit less events to reduce benchopera load
	// confirmingEmitInterval = piecefunc(totalStakeBeforeMe / totalStake) * MinEmitInterval
	myIdx := newValidators.GetIdx(em.config.Validator)
	totalStake := pos.Weight(0)
	totalStakeBeforeMe := pos.Weight(0)
	for i, stake := range newValidators.SortedStakes() {
		totalStake += stake
		if idx.Validator(i) < myIdx {
			totalStakeBeforeMe += stake
		}
	}
	stakeRatio := uint64((totalStakeBeforeMe * piecefunc.PercentUnit) / totalStake)
	confirmingEmitIntervalRatio := piecefunc.Get(stakeRatio, confirmingEmitIntervalPieces)
	em.intervals.Confirming = time.Duration(piecefunc.Mul(uint64(em.config.EmitIntervals.Confirming), confirmingEmitIntervalRatio))

	// validators with lower stake should emit more events at idle, to catch up with other validators if their frame is behind
	// MaxEmitInterval = piecefunc(totalStakeBeforeMe / totalStake) * MaxEmitInterval
	maxEmitIntervalRatio := piecefunc.Get(stakeRatio, maxEmitIntervalPieces)
	em.intervals.Max = time.Duration(piecefunc.Mul(uint64(em.config.EmitIntervals.Max), maxEmitIntervalRatio))

	// track when I've became validator
	now := time.Now()
	if em.config.Validator != 0 && !em.world.App.HasEpochValidator(newEpoch-1, em.config.Validator) {
		em.syncStatus.becameValidator = now
	}
}

// OnNewEvent tracks new events to find out am I properly synced or not
func (em *Emitter) OnNewEvent(e *inter.Event) {
	if em.config.Validator == 0 || em.config.Validator != e.Creator {
		return
	}
	if em.syncStatus.prevLocalEmittedID == e.Hash() {
		return
	}

	// event was emitted by me on another instance
	em.syncStatus.lastExternalSelfEvent = time.Now()

	eventTime := inter.MaxTimestamp(e.ClaimedTime, e.MedianTime).Time()
	if eventTime.Before(em.syncStatus.startup) {
		return
	}

	passedSinceEvent := time.Since(eventTime)
	threshold := em.intervals.SelfForkProtection
	if threshold > time.Minute {
		threshold = time.Minute
	}
	if passedSinceEvent <= threshold {
		reason := "Received a recent event (event id=%s) from this validator (validator ID=%d) which wasn't created on this node.\n" +
			"This external event was created %s, %s ago at the time of this error.\n" +
			"It might mean that a duplicating instance of the same validator is running simultaneously, which may eventually lead to a doublesign.\n" +
			"The node was stopped by one of the doublesign protection heuristics.\n" +
			"There's no guaranteed automatic protection against a doublesign," +
			"please always ensure that no more than one instance of the same validator is running."
		errlock.Permanent(fmt.Errorf(reason, e.Hash().String(), em.config.Validator, e.ClaimedTime.Time().Local().String(), passedSinceEvent.String()))
	}

}
func (em *Emitter) onNewExternalEvent(e *inter.Event) {
	// event was emitted by me on another instance
	em.syncStatus.lastExternalSelfEvent = time.Now()
	doublesign.DetectParallelInstance()
}

func (em *Emitter) getSyncStatus() doublesign.SyncStatus {
	return doublesign.SyncStatus{
		PeersNum:              em.world.PeersNum(),
		Now:                   time.Now(),
		Startup:               em.syncStatus.startup,
		LastConnected:         em.syncStatus.lastConnected,
		P2PSynced:             em.syncStatus.p2pSynced,
		BecameValidator:       em.syncStatus.becameValidator,
		LastSelfExternalEvent: em.syncStatus.lastExternalSelfEvent,
	}
}

func (em *Emitter) isSyncedToEmit() (time.Duration, error) {
	if em.intervals.SelfForkProtection == 0 {
		return 0, nil // protection disabled
	}
	return doublesign.SyncedToEmit(em.getSyncStatus(), em.intervals.SelfForkProtection)
}

func (em *Emitter) logSyncStatus(synced bool, reason string, wait time.Duration) bool {
	if synced {
		return true
	}

	if wait == 0 {
		em.Periodic.Info(25*time.Second, "Emitting is paused", "reason", reason)
	} else {
		em.Periodic.Info(25*time.Second, "Emitting is paused", "reason", reason, "wait", wait)
	}
	return false
}

// return true if event is in epoch tail (unlikely to confirm)
func (em *Emitter) isEpochTail(e *inter.Event) bool {
	return e.Frame() >= idx.Frame(em.net.Dag.MaxEpochBlocks)-em.config.EpochTailLength
}

func (em *Emitter) isAllowedToEmit(e *inter.Event, selfParent *inter.Event) bool {
	passedTime := e.CreationTime().Time().Sub(em.prevEmittedTime)
	// Slow down emitting if no payload to post, and not at epoch tail
	{
		if passedTime < em.intervals.Max &&
			len(e.Payload()) == 0 &&
			!em.isEpochTail(e) {
			return false
		}
	}

	return true
}

func (em *Emitter) EmitEvent() *inter.Event {
	if em.config.Validator == 0 {
		return nil // short circuit if not validator
	}

	poolTxs, err := em.world.Txpool.Pending() // request txs before locking engineMu to prevent deadlock!
	if err != nil {
		em.Log.Error("Tx pool transactions fetching error", "err", err)
		return nil
	}

	for _, tt := range poolTxs {
		for _, t := range tt {
			span := tracing.CheckTx(t.Hash(), "Emitter.EmitEvent(candidate)")
			defer span.Finish()
		}
	}

	em.world.EngineMu.Lock()
	defer em.world.EngineMu.Unlock()

	e := em.createEvent(poolTxs)
	if e == nil {
		return nil
	}
	em.syncStatus.prevLocalEmittedID = e.Hash()

	if em.world.OnEmitted != nil {
		em.world.OnEmitted(e)
	}
	em.gasRate.Mark(int64(e.GasPowerUsed))
	em.prevEmittedTime = time.Now() // record time after connecting, to add the event processing time"
	em.Log.Info("New event emitted", "id", e.Hash(), "parents", len(e.Parents), "by", e.Creator, "frame", inter.FmtFrame(e.Frame, e.IsRoot), "txs", e.Transactions.Len(), "t", time.Since(e.ClaimedTime.Time()))

	// metrics
	for _, t := range e.Transactions {
		span := tracing.CheckTx(t.Hash(), "Emitter.EmitEvent()")
		defer span.Finish()
	}

	return e
}

func (em *Emitter) nameEventForDebug(e *inter.Event) {
	name := []rune(hash.GetNodeName(e.Creator))
	if len(name) < 1 {
		return
	}

	name = name[len(name)-1:]
	hash.SetEventName(e.Hash(), fmt.Sprintf("%s%03d",
		strings.ToLower(string(name)),
		e.Seq))
}
