package app

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/Fantom-foundation/go-lachesis/inter/sfctype"
)

// SetSfcDelegation stores SfcDelegation
func (s *Store) SetSfcDelegation(id sfctype.DelegatorID, v *sfctype.SfcDelegation) {
	s.set(s.table.Delegators, id.Bytes(), v)

	// Add to LRU cache.
	if s.cache.Delegators != nil {
		s.cache.Delegators.Add(id, v)
	}
}

// DelSfcDelegation deletes SfcDelegation
func (s *Store) DelSfcDelegation(id sfctype.DelegatorID) {
	err := s.table.Delegators.Delete(id.Bytes())
	if err != nil {
		s.Log.Crit("Failed to erase delegator")
	}

	// Add to LRU cache.
	if s.cache.Delegators != nil {
		s.cache.Delegators.Remove(id)
	}
}

// ForEachSfcDelegation iterates all stored SfcDelegations
func (s *Store) ForEachSfcDelegation(do func(sfctype.SfcDelegationAndID)) {
	it := s.table.Delegators.NewIterator()
	defer it.Release()
	s.forEachSfcDelegation(it, func(id sfctype.SfcDelegationAndID) bool {
		do(id)
		return true
	})
}

// GetSfcDelegationsByAddr returns a lsit of delegations by address
func (s *Store) GetSfcDelegationsByAddr(addr common.Address, limit int) []sfctype.SfcDelegationAndID {
	it := s.table.Delegators.NewIteratorWithPrefix(addr.Bytes())
	defer it.Release()
	res := make([]sfctype.SfcDelegationAndID, 0, limit)
	s.forEachSfcDelegation(it, func(id sfctype.SfcDelegationAndID) bool {
		res = append(res, id)
		limit -= 1
		return limit == 0
	})
	return res
}

func (s *Store) forEachSfcDelegation(it ethdb.Iterator, do func(sfctype.SfcDelegationAndID) bool) {
	_continue := true
	for _continue && it.Next() {
		delegator := &sfctype.SfcDelegation{}
		err := rlp.DecodeBytes(it.Value(), delegator)
		if err != nil {
			s.Log.Crit("Failed to decode rlp while iteration", "err", err)
		}

		addr := it.Key()[len(it.Key())-sfctype.DelegatorIDSize:]
		_continue = do(sfctype.SfcDelegationAndID{
			ID:        sfctype.BytesToDelegatorID(addr),
			Delegator: delegator,
		})
	}
}

// GetSfcDelegation returns stored SfcDelegation
func (s *Store) GetSfcDelegation(id sfctype.DelegatorID) *sfctype.SfcDelegation {
	// Get data from LRU cache first.
	if s.cache.Delegators != nil {
		if c, ok := s.cache.Delegators.Get(id); ok {
			if b, ok := c.(*sfctype.SfcDelegation); ok {
				return b
			}
		}
	}

	w, _ := s.get(s.table.Delegators, id.Bytes(), &sfctype.SfcDelegation{}).(*sfctype.SfcDelegation)

	// Add to LRU cache.
	if w != nil && s.cache.Delegators != nil {
		s.cache.Delegators.Add(id, w)
	}

	return w
}
