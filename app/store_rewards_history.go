package app

import (
	"math/big"

	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/Fantom-foundation/go-lachesis/inter/sfctype"
)

// GetStakerClaimedRewards returns sum of claimed rewards in past, by this staker
func (s *Store) GetStakerClaimedRewards(stakerID idx.StakerID) *big.Int {
	amount, err := s.table.StakerOldRewards.Get(stakerID.Bytes())
	if err != nil {
		s.Log.Crit("Failed to erase key-value", "err", err)
	}
	if amount == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(amount)
}

// SetStakerClaimedRewards sets sum of claimed rewards in past
func (s *Store) SetStakerClaimedRewards(stakerID idx.StakerID, amount *big.Int) {
	err := s.table.StakerOldRewards.Put(stakerID.Bytes(), amount.Bytes())
	if err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// IncStakerClaimedRewards increments sum of claimed rewards in past
func (s *Store) IncStakerClaimedRewards(stakerID idx.StakerID, diff *big.Int) {
	amount := s.GetStakerClaimedRewards(stakerID)
	amount.Add(amount, diff)
	s.SetStakerClaimedRewards(stakerID, amount)
}

// DelStakerClaimedRewards deletes record about sum of claimed rewards in past
func (s *Store) DelStakerClaimedRewards(stakerID idx.StakerID) {
	err := s.table.StakerOldRewards.Delete(stakerID.Bytes())
	if err != nil {
		s.Log.Crit("Failed to erase key-value", "err", err)
	}
}

// GetDelegationClaimedRewards returns sum of claimed rewards in past, by this delegator
func (s *Store) GetDelegationClaimedRewards(id sfctype.DelegatorID) *big.Int {
	amount, err := s.table.DelegatorOldRewards.Get(id.Bytes())
	if err != nil {
		s.Log.Crit("Failed to erase key-value", "err", err)
	}
	if amount == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(amount)
}

// SetDelegatorClaimedRewards sets sum of claimed rewards in past
func (s *Store) SetDelegatorClaimedRewards(id sfctype.DelegatorID, amount *big.Int) {
	err := s.table.DelegatorOldRewards.Put(id.Bytes(), amount.Bytes())
	if err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// IncDelegatorClaimedRewards increments sum of claimed rewards in past
func (s *Store) IncDelegatorClaimedRewards(id sfctype.DelegatorID, diff *big.Int) {
	amount := s.GetDelegationClaimedRewards(id)
	amount.Add(amount, diff)
	s.SetDelegatorClaimedRewards(id, amount)
}

// DelDelegatorClaimedRewards deletes record about sum of claimed rewards in past
func (s *Store) DelDelegatorClaimedRewards(id sfctype.DelegatorID) {
	err := s.table.DelegatorOldRewards.Delete(id.Bytes())
	if err != nil {
		s.Log.Crit("Failed to erase key-value", "err", err)
	}
}

// GetStakerDelegationsClaimedRewards returns sum of claimed rewards in past, by this delegators of this staker
func (s *Store) GetStakerDelegationsClaimedRewards(stakerID idx.StakerID) *big.Int {
	amount, err := s.table.StakerDelegatorsOldRewards.Get(stakerID.Bytes())
	if err != nil {
		s.Log.Crit("Failed to erase key-value", "err", err)
	}
	if amount == nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(amount)
}

// SetStakerDelegatorsClaimedRewards sets sum of claimed rewards in past
func (s *Store) SetStakerDelegatorsClaimedRewards(stakerID idx.StakerID, amount *big.Int) {
	err := s.table.StakerDelegatorsOldRewards.Put(stakerID.Bytes(), amount.Bytes())
	if err != nil {
		s.Log.Crit("Failed to put key-value", "err", err)
	}
}

// IncStakerDelegatorsClaimedRewards increments sum of claimed rewards in past
func (s *Store) IncStakerDelegatorsClaimedRewards(stakerID idx.StakerID, diff *big.Int) {
	amount := s.GetStakerDelegationsClaimedRewards(stakerID)
	amount.Add(amount, diff)
	s.SetStakerDelegatorsClaimedRewards(stakerID, amount)
}

// DelStakerDelegatorsClaimedRewards deletes record about sum of claimed rewards in past
func (s *Store) DelStakerDelegatorsClaimedRewards(stakerID idx.StakerID) {
	err := s.table.StakerDelegatorsOldRewards.Delete(stakerID.Bytes())
	if err != nil {
		s.Log.Crit("Failed to erase key-value", "err", err)
	}
}
