package gossip

import (
	"fmt"
	"github.com/Fantom-foundation/go-lachesis/gossip/emitter"
	"github.com/Fantom-foundation/lachesis-base/eventcheck/epochcheck"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/lachesis"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	notify "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/Fantom-foundation/go-lachesis/ethapi"
	"github.com/Fantom-foundation/go-lachesis/eventcheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/basiccheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/heavycheck"
	"github.com/Fantom-foundation/go-lachesis/eventcheck/parentscheck"
	"github.com/Fantom-foundation/go-lachesis/evmcore"
	"github.com/Fantom-foundation/go-lachesis/gossip/filters"
	"github.com/Fantom-foundation/go-lachesis/inter"

	"github.com/Fantom-foundation/go-lachesis/benchopera"
	"github.com/Fantom-foundation/go-lachesis/logger"
)

const (
	txsRingBufferSize = 20000 // Maximum number of stored hashes of included but not confirmed txs
)

type ServiceFeed struct {
	scope notify.SubscriptionScope

	newEpoch        notify.Feed
	newPack         notify.Feed
	newEmittedEvent notify.Feed
	newBlock        notify.Feed
	newTxs          notify.Feed
	newLogs         notify.Feed
}

func (f *ServiceFeed) SubscribeNewEpoch(ch chan<- idx.Epoch) notify.Subscription {
	return f.scope.Track(f.newEpoch.Subscribe(ch))
}

func (f *ServiceFeed) SubscribeNewPack(ch chan<- idx.Pack) notify.Subscription {
	return f.scope.Track(f.newPack.Subscribe(ch))
}

func (f *ServiceFeed) SubscribeNewEmitted(ch chan<- *inter.Event) notify.Subscription {
	return f.scope.Track(f.newEmittedEvent.Subscribe(ch))
}

func (f *ServiceFeed) SubscribeNewLogs(ch chan<- []*types.Log) notify.Subscription {
	return f.scope.Track(f.newLogs.Subscribe(ch))
}

// Service implements go-ethereum/node.Service interface.
type Service struct {
	config *Config

	wg   sync.WaitGroup
	done chan struct{}

	// server
	Name  string
	Topic discv5.Topic

	serverPool *serverPool

	// application
	node             *node.ServiceContext
	store            *Store
	engine           lachesis.Consensus
	engineMu         *sync.RWMutex
	emitter          *emitter.Emitter
	heavyCheckReader HeavyCheckReader
	checkers         *eventcheck.Checkers

	feed ServiceFeed

	// application protocol
	pm *ProtocolManager

	API           *APIBackend
	netRPCService *ethapi.PublicNetAPI

	stopped bool

	logger.Instance
}

func NewService(ctx *node.ServiceContext, config *Config, store *Store, engine lachesis.Consensus) (*Service, error) {
	svc := &Service{
		config: config,

		done: make(chan struct{}),

		Name: fmt.Sprintf("Node-%d", rand.Int()),

		node:  ctx,
		store: store,

		engine:   engine,
		engineMu: new(sync.RWMutex),

		Instance: logger.MakeInstance(),
	}

	// create server pool
	trustedNodes := []string{}
	svc.serverPool = newServerPool(store.async.table.Peers, svc.done, &svc.wg, trustedNodes)

	// create checkers
	svc.heavyCheckReader.PubKeys.Store(NewEpochPubKeys(svc.store.GetEpoch(), svc.config.Net.Genesis.Validators))                                                                     // read pub keys of current epoch from disk
	svc.checkers = makeCheckers(&svc.config.Net, &svc.heavyCheckReader, svc.store)

	// create protocol manager
	var err error
	svc.pm, err = NewProtocolManager(config, &svc.feed, svc.engineMu, svc.checkers, store, svc.processEvent, svc.serverPool)

	// create API backend
	svc.API = &APIBackend{config.ExtRPCEnabled, svc, stateReader, nil}

	return svc, err
}

// makeCheckers builds event checkers
func makeCheckers(net *benchopera.Config, heavyCheckReader *HeavyCheckReader, store *Store) *eventcheck.Checkers {
	// create signatures checker
	heavyCheck := heavycheck.NewDefault(&net.Dag, heavyCheckReader)

	return &eventcheck.Checkers{
		Basiccheck:   basiccheck.New(&net.Dag),
		Epochcheck:   epochcheck.New(store),
		Parentscheck: parentscheck.New(),
		Heavycheck:   heavyCheck,
	}
}

func (s *Service) makeEmitter() *emitter.Emitter {
	// randomize event time to decrease peak load, and increase chance of catching double instances of validator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	emitterCfg := s.config.Emitter // copy data
	emitterCfg.EmitIntervals = *emitterCfg.EmitIntervals.RandomizeEmitTime(r)

	return emitter.NewEmitter(&s.config.Net, nil, &emitterCfg,
		emitter.EmitterWorld{
			Store:    s.store,
			EngineMu: s.engineMu,
			ProcessEvent: func(emitted *inter.Event) error {
				// s.engineMu is locked here

				err := s.processEvent(emitted)
				if err != nil {
					s.Log.Crit("Self-event connection failed", "err", err.Error())
				}

				s.feed.newEmittedEvent.Send(emitted) // PM listens and will broadcast it
				if err != nil {
					s.Log.Crit("Failed to post self-event", "err", err.Error())
				}
				return nil
			},
			Build: func(event dag.MutableEvent) error {
				return s.engine.Build(event)
			},
			IsSynced: func() bool {
				return atomic.LoadUint32(&s.pm.synced) != 0
			},
			PeersNum: func() int {
				return s.pm.peers.Len()
			},
			Checkers: s.checkers,
		},
	)
}

// Protocols returns protocols the service can communicate on.
func (s *Service) Protocols() []p2p.Protocol {
	protos := make([]p2p.Protocol, len(ProtocolVersions))
	for i, vsn := range ProtocolVersions {
		protos[i] = s.pm.makeProtocol(vsn)
		protos[i].Attributes = []enr.Entry{s.currentEnr()}
	}
	return protos
}

// APIs returns api methods the service wants to expose on rpc channels.
func (s *Service) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.API)

	apis = append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.API),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)

	return apis
}

// Start method invoked when the node is ready to start the service.
func (s *Service) Start(srv *p2p.Server) error {
	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srv, s.config.Net.NetworkID)

	genesis := s.store.GetBlock(0).Atropos
	s.Topic = discv5.Topic("benchopera@" + genesis.Hex())

	if srv.DiscV5 != nil {
		go func(topic discv5.Topic) {
			s.Log.Info("Starting topic registration")
			defer s.Log.Info("Terminated topic registration")

			srv.DiscV5.RegisterTopic(topic, s.done)
		}(s.Topic)
	}

	s.pm.Start(srv.MaxPeers)

	s.serverPool.start(srv, s.Topic)

	s.emitter = s.makeEmitter()
	s.emitter.StartEventEmission()

	return nil
}

// Stop method invoked when the node terminates the service.
func (s *Service) Stop() error {
	close(s.done)
	s.emitter.StopEventEmission()
	s.pm.Stop()
	s.wg.Wait()
	s.feed.scope.Close()

	// flush the state at exit, after all the routines stopped
	s.engineMu.Lock()
	defer s.engineMu.Unlock()
	s.stopped = true

	return s.store.Commit(nil, true)
}

// AccountManager return node's account manager
func (s *Service) AccountManager() *accounts.Manager {
	return s.node.AccountManager
}
