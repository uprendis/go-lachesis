package integration

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/dag"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/utils/adapters"
	"github.com/Fantom-foundation/lachesis-base/vector"

	"github.com/Fantom-foundation/lachesis-base/abft"
	"github.com/Fantom-foundation/lachesis-base/kvdb/flushable"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/Fantom-foundation/go-lachesis/gossip"
)

func panics(name string) func(error) {
	return func(err error) {
		log.Crit(fmt.Sprintf("%s error", name), "err", err)
	}
}

type GossipStoreAdapter struct {
	*gossip.Store
}

func (g *GossipStoreAdapter) GetEvent(id hash.Event) dag.Event {
	return g.Store.GetEvent(id)
}

// MakeEngine makes consensus engine from config.
func MakeEngine(dataDir string, gossipCfg *gossip.Config) (*abft.Lachesis, *flushable.SyncedPool, *gossip.Store) {
	dbs := flushable.NewSyncedPool(DBProducer(dataDir))

	gdb := gossip.NewStore(dbs, gossipCfg.StoreConfig)

	cMainDb := dbs.GetDb("benchopera")
	cGetEpochDB := func(epoch idx.Epoch) kvdb.DropableStore {
		return dbs.GetDb(fmt.Sprintf("benchopera-%d", epoch))
	}
	cdb := abft.NewStore(cMainDb, cGetEpochDB, panics("Lachesis store"), abft.DefaultStoreConfig())

	// write genesis

	err := gdb.Migrate()
	if err != nil {
		utils.Fatalf("Failed to migrate Gossip DB: %v", err)
	}
	genesisAtropos, _, isNew, err := gdb.ApplyGenesis(&gossipCfg.Net)
	if err != nil {
		utils.Fatalf("Failed to write Gossip genesis state: %v", err)
	}

	if isNew {
		err = cdb.ApplyGenesis(&abft.Genesis{
			Validators: nil,
			Atropos:    genesisAtropos,
		})
		if err != nil {
			utils.Fatalf("Failed to write Lachesis genesis state: %v", err)
		}
	}

	err = dbs.Flush(genesisAtropos.Bytes())
	if err != nil {
		utils.Fatalf("Failed to flush genesis state: %v", err)
	}

	if isNew {
		log.Info("Applied genesis state", "hash", genesisAtropos.FullID())
	} else {
		log.Info("Genesis state is already written", "hash", genesisAtropos.FullID())
	}

	// create consensus
	vecClock := vector.NewIndex(panics("Vector clock"), vector.DefaultConfig())
	engine := abft.NewLachesis(cdb, &GossipStoreAdapter{gdb}, &adapters.VectorToDagIndexer{vecClock}, panics("Lachesis"), abft.DefaultConfig())

	return engine, dbs, gdb
}

// SetAccountKey sets key into accounts manager and unlocks it with pswd.
func SetAccountKey(
	am *accounts.Manager, key *ecdsa.PrivateKey, pswd string,
) (
	acc accounts.Account,
) {
	kss := am.Backends(keystore.KeyStoreType)
	if len(kss) < 1 {
		log.Warn("Keystore is not found")
		return
	}
	ks := kss[0].(*keystore.KeyStore)

	acc = accounts.Account{
		Address: crypto.PubkeyToAddress(key.PublicKey),
	}

	imported, err := ks.ImportECDSA(key, pswd)
	if err == nil {
		acc = imported
	} else if err.Error() != "account already exists" {
		log.Crit("Failed to import key", "err", err)
	}

	err = ks.Unlock(acc, pswd)
	if err != nil {
		log.Crit("failed to unlock key", "err", err)
	}

	return
}
