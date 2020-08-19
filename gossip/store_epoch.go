package gossip

/*
	In LRU cache data stored like pointer
*/

import (
	"errors"
	"github.com/Fantom-foundation/lachesis-base/hash"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/Fantom-foundation/lachesis-base/kvdb"
	"github.com/Fantom-foundation/lachesis-base/kvdb/skiperrors"
	"github.com/Fantom-foundation/lachesis-base/kvdb/table"
)

type (
	epochStore struct {
		Tips    kvdb.Store `table:"t"`
		Heads   kvdb.Store `table:"H"`
	}
)

func newEpochStore(db kvdb.Store) *epochStore {
	es := &epochStore{}
	table.MigrateTables(es, db)

	err := errors.New("database closed")

	es.Tips = skiperrors.Wrap(es.Tips, err)
	es.Heads = skiperrors.Wrap(es.Heads, err)

	return es
}

// getEpochStore is not safe for concurrent use.
func (s *Store) getEpochStore(epoch idx.Epoch) *epochStore {
	tables := s.EpochDbs.Get(uint64(epoch))
	if tables == nil {
		return nil
	}

	return tables.(*epochStore)
}

// delEpochStore is not safe for concurrent use.
func (s *Store) delEpochStore(epoch idx.Epoch) {
	s.EpochDbs.Del(uint64(epoch))
}

// SetLastEvent stores last unconfirmed event from a validator (off-chain)
func (s *Store) SetLastEvent(epoch idx.Epoch, from idx.ValidatorID, id hash.Event) {
	es := s.getEpochStore(epoch)
	if es == nil {
		return
	}

	key := from.Bytes()
	if err := es.Tips.Put(key, id.Bytes()); err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// GetLastEvent returns stored last unconfirmed event from a validator (off-chain)
func (s *Store) GetLastEvent(from idx.ValidatorID) *hash.Event {
	es := s.getEpochStore(s.GetEpoch())
	if es == nil {
		return nil
	}

	key := from.Bytes()
	idBytes, err := es.Tips.Get(key)
	if err != nil {
		s.Log.Crit("Failed to get key-value", "err", err)
	}
	if idBytes == nil {
		return nil
	}
	id := hash.BytesToEvent(idBytes)
	return &id
}
