package gossip

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/Fantom-foundation/go-lachesis/app"
	"github.com/Fantom-foundation/go-lachesis/inter"

	"github.com/Fantom-foundation/go-lachesis/inter/sfctype"
	"github.com/Fantom-foundation/go-lachesis/benchopera"
	"github.com/Fantom-foundation/go-lachesis/benchopera/genesis/sfc"
	"github.com/Fantom-foundation/go-lachesis/benchopera/genesis/sfc/sfcpos"
	"github.com/Fantom-foundation/go-lachesis/utils"
)

// GetActiveSfcValidators returns validators which will become validators in next epoch
func (s *Service) GetActiveSfcValidators() []sfctype.SfcValidatorAndID {
	validators := make([]sfctype.SfcValidatorAndID, 0, 200)
	s.app.ForEachSfcValidator(func(it sfctype.SfcValidatorAndID) {
		if it.Validator.Ok() {
			validators = append(validators, it)
		}
	})
	return validators
}

func (s *Service) delAllValidatorData(validatorID idx.ValidatorID) {
	s.app.DelSfcValidator(validatorID)
	s.app.ResetBlocksMissed(validatorID)
	s.app.DelActiveValidationScore(validatorID)
	s.app.DelDirtyValidationScore(validatorID)
	s.app.DelActiveOriginationScore(validatorID)
	s.app.DelDirtyOriginationScore(validatorID)
	s.app.DelWeightedDelegationsFee(validatorID)
	s.app.DelValidatorPOI(validatorID)
	s.app.DelValidatorClaimedRewards(validatorID)
	s.app.DelValidatorDelegationsClaimedRewards(validatorID)
}

func (s *Service) delAllDelegationData(id sfctype.DelegationID) {
	s.app.DelSfcDelegation(id)
	s.app.DelDelegationClaimedRewards(id)
}

var (
	max128 = new(big.Int).Sub(math.BigPow(2, 128), common.Big1)
)

func (s *Service) calcRewardWeights(validators []sfctype.SfcValidatorAndID, _epochDuration inter.Timestamp) (baseRewardWeights []*big.Int, txRewardWeights []*big.Int) {
	validationScores := make([]*big.Int, 0, len(validators))
	originationScores := make([]*big.Int, 0, len(validators))
	pois := make([]*big.Int, 0, len(validators))
	stakes := make([]*big.Int, 0, len(validators))

	if _epochDuration == 0 {
		_epochDuration = 1
	}
	epochDuration := new(big.Int).SetUint64(uint64(_epochDuration))

	for _, it := range validators {
		stake := it.Validator.CalcTotalStake()
		poi := s.app.GetValidatorPOI(it.ValidatorID)
		validationScore := s.app.GetActiveValidationScore(it.ValidatorID)
		originationScore := s.app.GetActiveOriginationScore(it.ValidatorID)

		stakes = append(stakes, stake)
		validationScores = append(validationScores, validationScore)
		originationScores = append(originationScores, originationScore)
		pois = append(pois, poi)
	}

	txRewardWeights = make([]*big.Int, 0, len(validators))
	for i := range validators {
		// txRewardWeight = ({origination score} + {CONST} * {PoI}) * {validation score}
		// origination score is roughly proportional to {validation score} * {stake}, so the whole formula is roughly
		// {stake} * {validation score} ^ 2
		poiWithRatio := new(big.Int).Mul(pois[i], s.config.Net.Economy.TxRewardPoiImpact)
		poiWithRatio.Div(poiWithRatio, benchopera.PercentUnit)

		txRewardWeight := new(big.Int).Add(originationScores[i], poiWithRatio)
		txRewardWeight.Mul(txRewardWeight, validationScores[i])
		txRewardWeight.Div(txRewardWeight, epochDuration)
		if txRewardWeight.Cmp(max128) > 0 {
			txRewardWeight = new(big.Int).Set(max128) // never going to get here
		}

		txRewardWeights = append(txRewardWeights, txRewardWeight)
	}

	baseRewardWeights = make([]*big.Int, 0, len(validators))
	for i := range validators {
		// baseRewardWeight = {stake} * {validationScore ^ 2}
		baseRewardWeight := new(big.Int).Set(stakes[i])
		for pow := 0; pow < 2; pow++ {
			baseRewardWeight.Mul(baseRewardWeight, validationScores[i])
			baseRewardWeight.Div(baseRewardWeight, epochDuration)
		}
		if baseRewardWeight.Cmp(max128) > 0 {
			baseRewardWeight = new(big.Int).Set(max128) // never going to get here
		}

		baseRewardWeights = append(baseRewardWeights, baseRewardWeight)
	}

	return baseRewardWeights, txRewardWeights
}

// getRewardPerSec returns current rewardPerSec, depending on config and value provided by SFC
func (s *Service) getRewardPerSec() *big.Int {
	rewardPerSecond := s.app.GetSfcConstants(s.engine.GetEpoch() - 1).BaseRewardPerSec
	if rewardPerSecond == nil || rewardPerSecond.Sign() == 0 {
		rewardPerSecond = s.config.Net.Economy.InitialRewardPerSecond
	}
	if rewardPerSecond.Cmp(s.config.Net.Economy.MaxRewardPerSecond) > 0 {
		rewardPerSecond = s.config.Net.Economy.MaxRewardPerSecond
	}
	return new(big.Int).Set(rewardPerSecond)
}

// getOfflinePenaltyThreshold returns current offlinePenaltyThreshold, depending on config and value provided by SFC
func (s *Service) getOfflinePenaltyThreshold() app.BlocksMissed {
	v := s.app.GetSfcConstants(s.engine.GetEpoch() - 1).OfflinePenaltyThreshold
	if v.Num == 0 {
		v.Num = s.config.Net.Economy.InitialOfflinePenaltyThreshold.BlocksNum
	}
	if v.Period == 0 {
		v.Period = inter.Timestamp(s.config.Net.Economy.InitialOfflinePenaltyThreshold.Period)
	}
	return v
}

// processSfc applies the new SFC state
func (s *Service) processSfc(block *inter.Block, receipts types.Receipts, blockFee *big.Int, sealEpoch bool, cheaters inter.Cheaters, statedb *state.StateDB) {
	// s.engineMu is locked here

	// process SFC contract logs
	epoch := s.engine.GetEpoch()
	for _, receipt := range receipts {
		for _, l := range receipt.Logs {
			if l.Address != sfc.ContractAddress {
				continue
			}
			// Add new validators
			if l.Topics[0] == sfcpos.Topics.CreatedStake && len(l.Topics) > 2 && len(l.Data) >= 32 {
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())
				address := common.BytesToAddress(l.Topics[2][12:])
				amount := new(big.Int).SetBytes(l.Data[0:32])

				s.app.SetSfcValidator(validatorID, &sfctype.SfcValidator{
					Address:      address,
					CreatedEpoch: epoch,
					CreatedTime:  block.Time,
					StakeAmount:  amount,
					DelegatedMe:  big.NewInt(0),
				})
			}

			// Increase stakes
			if l.Topics[0] == sfcpos.Topics.IncreasedStake && len(l.Topics) > 1 && len(l.Data) >= 32 {
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())
				newAmount := new(big.Int).SetBytes(l.Data[0:32])

				validator := s.app.GetSfcValidator(validatorID)
				if validator == nil {
					s.Log.Warn("Internal SFC index isn't synced with SFC contract")
					continue
				}
				validator.StakeAmount = newAmount
				s.app.SetSfcValidator(validatorID, validator)
			}

			// Add new delegations
			if l.Topics[0] == sfcpos.Topics.CreatedDelegation && len(l.Topics) > 1 && len(l.Data) >= 32 {
				address := common.BytesToAddress(l.Topics[1][12:])
				toValidatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[2][:]).Uint64())
				amount := new(big.Int).SetBytes(l.Data[0:32])

				validator := s.app.GetSfcValidator(toValidatorID)
				if validator == nil {
					s.Log.Warn("Internal SFC index isn't synced with SFC contract")
					continue
				}
				validator.DelegatedMe.Add(validator.DelegatedMe, amount)

				s.app.SetSfcDelegation(sfctype.DelegationID{address, toValidatorID}, &sfctype.SfcDelegation{
					CreatedEpoch: epoch,
					CreatedTime:  block.Time,
					Amount:       amount,
				})
				s.app.SetSfcValidator(toValidatorID, validator)
			}

			// Deactivate stakes
			if (l.Topics[0] == sfcpos.Topics.DeactivatedStake || l.Topics[0] == sfcpos.Topics.PreparedToWithdrawStake) && len(l.Topics) > 1 {
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())

				validator := s.app.GetSfcValidator(validatorID)
				validator.DeactivatedEpoch = epoch
				validator.DeactivatedTime = block.Time
				s.app.SetSfcValidator(validatorID, validator)
			}

			// Deactivate delegations
			if (l.Topics[0] == sfcpos.Topics.DeactivatedDelegation || l.Topics[0] == sfcpos.Topics.PreparedToWithdrawDelegation) && len(l.Topics) > 2 {
				address := common.BytesToAddress(l.Topics[1][12:])
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[2][:]).Uint64())
				id := sfctype.DelegationID{address, validatorID}

				delegation := s.app.GetSfcDelegation(id)
				validator := s.app.GetSfcValidator(validatorID)
				if validator != nil {
					validator.DelegatedMe.Sub(validator.DelegatedMe, delegation.Amount)
					if validator.DelegatedMe.Sign() < 0 {
						validator.DelegatedMe = big.NewInt(0)
					}
					s.app.SetSfcValidator(validatorID, validator)
				}
				delegation.DeactivatedEpoch = epoch
				delegation.DeactivatedTime = block.Time
				s.app.SetSfcDelegation(id, delegation)
			}

			// Update stake
			if l.Topics[0] == sfcpos.Topics.UpdatedStake && len(l.Topics) > 1 && len(l.Data) >= 64 {
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())
				newAmount := new(big.Int).SetBytes(l.Data[0:32])
				newDelegatedMe := new(big.Int).SetBytes(l.Data[32:64])

				validator := s.app.GetSfcValidator(validatorID)
				if validator == nil {
					s.Log.Warn("Internal SFC index isn't synced with SFC contract")
					continue
				}
				validator.StakeAmount = newAmount
				validator.DelegatedMe = newDelegatedMe
				s.app.SetSfcValidator(validatorID, validator)
			}

			// Update delegation
			if l.Topics[0] == sfcpos.Topics.UpdatedDelegation && len(l.Topics) > 3 && len(l.Data) >= 32 {
				address := common.BytesToAddress(l.Topics[1][12:])
				oldValidatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[3][:]).Uint64())
				newValidatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[3][:]).Uint64())
				newAmount := new(big.Int).SetBytes(l.Data[0:32])
				oldId := sfctype.DelegationID{address, oldValidatorID}
				newId := sfctype.DelegationID{address, newValidatorID}

				delegation := s.app.GetSfcDelegation(oldId)
				if delegation == nil {
					s.Log.Warn("Internal SFC index isn't synced with SFC contract")
					continue
				}
				delegation.Amount = newAmount
				s.app.DelSfcDelegation(oldId)
				s.app.SetSfcDelegation(newId, delegation)
			}

			// Delete stakes
			if l.Topics[0] == sfcpos.Topics.WithdrawnStake && len(l.Topics) > 1 {
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())
				s.delAllValidatorData(validatorID)
			}

			// Delete delegations
			if l.Topics[0] == sfcpos.Topics.WithdrawnDelegation && len(l.Topics) > 2 {
				address := common.BytesToAddress(l.Topics[1][12:])
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[2][:]).Uint64())
				s.delAllDelegationData(sfctype.DelegationID{address, validatorID})
			}

			// Track changes of constants by SFC
			if l.Topics[0] == sfcpos.Topics.UpdatedBaseRewardPerSec && len(l.Data) >= 32 {
				baseRewardPerSec := new(big.Int).SetBytes(l.Data[0:32])
				constants := s.app.GetSfcConstants(epoch)
				constants.BaseRewardPerSec = baseRewardPerSec
				s.app.SetSfcConstants(epoch, constants)
			}
			if l.Topics[0] == sfcpos.Topics.UpdatedGasPowerAllocationRate && len(l.Data) >= 64 {
				shortAllocationRate := new(big.Int).SetBytes(l.Data[0:32])
				longAllocationRate := new(big.Int).SetBytes(l.Data[32:64])
				constants := s.app.GetSfcConstants(epoch)
				constants.ShortGasPowerAllocPerSec = shortAllocationRate.Uint64()
				constants.LongGasPowerAllocPerSec = longAllocationRate.Uint64()
				s.app.SetSfcConstants(epoch, constants)
			}
			if l.Topics[0] == sfcpos.Topics.UpdatedOfflinePenaltyThreshold && len(l.Data) >= 64 {
				blocksNum := new(big.Int).SetBytes(l.Data[0:32])
				period := new(big.Int).SetBytes(l.Data[32:64])
				constants := s.app.GetSfcConstants(epoch)
				constants.OfflinePenaltyThreshold.Num = idx.Block(blocksNum.Uint64())
				constants.OfflinePenaltyThreshold.Period = inter.Timestamp(period.Uint64())
				s.app.SetSfcConstants(epoch, constants)
			}

			// Track rewards (API-only)
			if l.Topics[0] == sfcpos.Topics.ClaimedValidatorReward && len(l.Topics) > 1 && len(l.Data) >= 32 {
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[1][:]).Uint64())
				reward := new(big.Int).SetBytes(l.Data[0:32])

				s.app.IncValidatorClaimedRewards(validatorID, reward)
			}
			if l.Topics[0] == sfcpos.Topics.ClaimedDelegationReward && len(l.Topics) > 2 && len(l.Data) >= 32 {
				address := common.BytesToAddress(l.Topics[1][12:])
				validatorID := idx.ValidatorID(new(big.Int).SetBytes(l.Topics[2][:]).Uint64())
				reward := new(big.Int).SetBytes(l.Data[0:32])

				s.app.IncDelegationClaimedRewards(sfctype.DelegationID{address, validatorID}, reward)
				s.app.IncValidatorDelegationsClaimedRewards(validatorID, reward)
			}
		}
	}

	// Update EpochStats
	stats := s.store.GetDirtyEpochStats()
	stats.TotalFee = new(big.Int).Add(stats.TotalFee, blockFee)
	if sealEpoch {
		// dirty EpochStats becomes active
		stats.End = block.Time
		s.store.SetEpochStats(epoch, stats)

		// new dirty EpochStats
		s.store.SetDirtyEpochStats(&sfctype.EpochStats{
			Start:    block.Time,
			TotalFee: new(big.Int),
		})
	} else {
		s.store.SetDirtyEpochStats(stats)
	}

	// Write cheaters
	for _, validatorID := range cheaters {
		validator := s.app.GetSfcValidator(validatorID)
		if validator.HasFork() {
			continue
		}
		// write into DB
		validator.Status |= sfctype.ForkBit
		s.app.SetSfcValidator(validatorID, validator)
		// write into SFC contract
		position := sfcpos.Validator(validatorID)
		statedb.SetState(sfc.ContractAddress, position.Status(), utils.U64to256(validator.Status))
	}

	if sealEpoch {
		if s.app.HasSfcConstants(epoch) {
			s.app.SetSfcConstants(epoch+1, s.app.GetSfcConstants(epoch))
		}

		// Write offline validators
		for _, it := range s.app.GetSfcValidators() {
			if it.Validator.Offline() {
				continue
			}

			gotMissed := s.app.GetBlocksMissed(it.ValidatorID)
			badMissed := s.getOfflinePenaltyThreshold()
			if gotMissed.Num >= badMissed.Num && gotMissed.Period >= badMissed.Period {
				// write into DB
				it.Validator.Status |= sfctype.OfflineBit
				s.app.SetSfcValidator(it.ValidatorID, it.Validator)
				// write into SFC contract
				position := sfcpos.Validator(it.ValidatorID)
				statedb.SetState(sfc.ContractAddress, position.Status(), utils.U64to256(it.Validator.Status))
			}
		}

		// Write epoch snapshot (for reward)
		cheatersSet := cheaters.Set()
		epochPos := sfcpos.EpochSnapshot(epoch)
		epochValidators := s.app.GetEpochValidators(epoch)
		baseRewardWeights, txRewardWeights := s.calcRewardWeights(epochValidators, stats.Duration())

		totalBaseRewardWeight := new(big.Int)
		totalTxRewardWeight := new(big.Int)
		totalStake := new(big.Int)
		totalDelegated := new(big.Int)
		for i, it := range epochValidators {
			baseRewardWeight := baseRewardWeights[i]
			txRewardWeight := txRewardWeights[i]
			totalStake.Add(totalStake, it.Validator.StakeAmount)
			totalDelegated.Add(totalDelegated, it.Validator.DelegatedMe)

			if _, ok := cheatersSet[it.ValidatorID]; ok {
				continue // don't give reward to cheaters
			}
			if baseRewardWeight.Sign() == 0 && txRewardWeight.Sign() == 0 {
				continue // don't give reward to offline validators
			}

			meritPos := epochPos.ValidatorMerit(it.ValidatorID)

			statedb.SetState(sfc.ContractAddress, meritPos.StakeAmount(), utils.BigTo256(it.Validator.StakeAmount))
			statedb.SetState(sfc.ContractAddress, meritPos.DelegatedMe(), utils.BigTo256(it.Validator.DelegatedMe))
			statedb.SetState(sfc.ContractAddress, meritPos.BaseRewardWeight(), utils.BigTo256(baseRewardWeight))
			statedb.SetState(sfc.ContractAddress, meritPos.TxRewardWeight(), utils.BigTo256(txRewardWeight))

			totalBaseRewardWeight.Add(totalBaseRewardWeight, baseRewardWeight)
			totalTxRewardWeight.Add(totalTxRewardWeight, txRewardWeight)
		}
		baseRewardPerSec := s.getRewardPerSec()

		// set total supply
		baseRewards := new(big.Int).Mul(big.NewInt(stats.Duration().Unix()), baseRewardPerSec)
		rewards := new(big.Int).Add(baseRewards, stats.TotalFee)
		totalSupply := new(big.Int).Add(s.app.GetTotalSupply(), rewards)
		statedb.SetState(sfc.ContractAddress, sfcpos.CurrentSealedEpoch(), utils.U64to256(uint64(epoch)))
		s.app.SetTotalSupply(totalSupply)

		statedb.SetState(sfc.ContractAddress, epochPos.TotalBaseRewardWeight(), utils.BigTo256(totalBaseRewardWeight))
		statedb.SetState(sfc.ContractAddress, epochPos.TotalTxRewardWeight(), utils.BigTo256(totalTxRewardWeight))
		statedb.SetState(sfc.ContractAddress, epochPos.EpochFee(), utils.BigTo256(stats.TotalFee))
		statedb.SetState(sfc.ContractAddress, epochPos.EndTime(), utils.U64to256(uint64(stats.End.Unix())))
		statedb.SetState(sfc.ContractAddress, epochPos.Duration(), utils.U64to256(uint64(stats.Duration().Unix())))
		statedb.SetState(sfc.ContractAddress, epochPos.BaseRewardPerSecond(), utils.BigTo256(baseRewardPerSec))
		statedb.SetState(sfc.ContractAddress, epochPos.StakeTotalAmount(), utils.BigTo256(totalStake))
		statedb.SetState(sfc.ContractAddress, epochPos.DelegationsTotalAmount(), utils.BigTo256(totalDelegated))
		statedb.SetState(sfc.ContractAddress, epochPos.TotalSupply(), utils.BigTo256(totalSupply))
		statedb.SetState(sfc.ContractAddress, sfcpos.CurrentSealedEpoch(), utils.U64to256(uint64(epoch)))

		// Add balance for SFC to pay rewards
		statedb.AddBalance(sfc.ContractAddress, rewards)

		// Select new validators
		s.app.SetEpochValidators(epoch+1, s.GetActiveSfcValidators())
	}
}
