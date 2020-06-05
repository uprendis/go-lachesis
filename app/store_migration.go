package app

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/Fantom-foundation/go-lachesis/evmcore"
	"github.com/Fantom-foundation/go-lachesis/hash"
	"github.com/Fantom-foundation/go-lachesis/inter"
	"github.com/Fantom-foundation/go-lachesis/inter/idx"
	"github.com/Fantom-foundation/go-lachesis/kvdb"
	"github.com/Fantom-foundation/go-lachesis/kvdb/flushable"
	"github.com/Fantom-foundation/go-lachesis/kvdb/table"
	"github.com/Fantom-foundation/go-lachesis/utils/migration"
)

func (s *Store) migrate() {
	versions := kvdb.NewIDStore(s.table.Version)
	err := s.migrations(s.dbs).Exec(versions)
	if err != nil {
		s.Log.Crit("app store migrations", "err", err)
	}
}

func (s *Store) migrations(dbs *flushable.SyncedPool) *migration.Migration {
	return migration.Begin("lachesis-app-store").
		Next("dedicated app-main database", func() (err error) {
			defer func() {
				if err == nil {
					s.Log.Warn("dedicated app-main database migration has been applied")
				}
			}()

			// NOTE: cross db dependency
			consensus := dbs.GetDb("gossip-main")
			engine := dbs.GetDb("poset-main")

			var src, dst tablesToMoveFromGossip
			table.MigrateTables(&src, consensus)
			table.MigrateTables(&dst, s.mainDb)

			for _, t := range [][2]kvdb.KeyValueStore{
				{src.Genesis, dst.Genesis},
				{src.ActiveValidationScore, dst.ActiveValidationScore},
				{src.DirtyValidationScore, dst.DirtyValidationScore},
				{src.ActiveOriginationScore, dst.ActiveOriginationScore},
				{src.DirtyOriginationScore, dst.DirtyOriginationScore},
				{src.BlockDowntime, dst.BlockDowntime},
				{src.StakerPOIScore, dst.StakerPOIScore},
				{src.AddressPOIScore, dst.AddressPOIScore},
				{src.AddressFee, dst.AddressFee},
				{src.StakerDelegatorsFee, dst.StakerDelegatorsFee},
				{src.AddressLastTxTime, dst.AddressLastTxTime},
				{src.TotalPoiFee, dst.TotalPoiFee},
				{src.Validators, dst.Validators},
				{src.Stakers, dst.Stakers},
				{src.Delegators, dst.Delegators},
				{src.SfcConstants, dst.SfcConstants},
				{src.TotalSupply, dst.TotalSupply},
				{src.Receipts, dst.Receipts},
				{src.DelegatorOldRewards, dst.DelegatorOldRewards},
				{src.StakerOldRewards, dst.StakerOldRewards},
				{src.StakerDelegatorsOldRewards, dst.StakerDelegatorsOldRewards},
				{src.ForEvmTable, dst.ForEvmTable},
				{src.ForEvmLogsTable, dst.ForEvmLogsTable},
				{src.EpochStats, dst.EpochStats},
			} {
				err = kvdb.Move(t[0], t[1], nil)
				if err != nil {
					return
				}
			}

			checkpoints := table.New(engine, []byte("c")) // table.Checkpoint
			cp, _ := s.get(checkpoints, []byte("c"), &engineCheckpoint{}).(*engineCheckpoint)
			if cp == nil {
				return
			}
			lastBlock := cp.LastBlockN - idx.Block(cp.LastDecidedFrame)

			blocks := table.New(consensus, []byte("b")) // table.Blocks
			b, _ := s.get(blocks, lastBlock.Bytes(), &inter.Block{}).(*inter.Block)
			if b == nil {
				return
			}

			s.SetCheckpoint(Checkpoint{
				BlockN:     cp.LastBlockN,
				EpochN:     cp.LastAtropos.Epoch(),
				EpochBlock: b.Index,
				EpochStart: b.Time,
			})

			return

		}).
		Next("app's own blocks", func() (err error) {
			defer func() {
				if err == nil {
					s.Log.Warn("app's own blocks migration has been applied")
				}
			}()

			// NOTE: cross db dependency
			consensus := dbs.GetDb("gossip-main")

			src := table.New(consensus, []byte("b")) // table.Blocks
			dst := table.New(s.mainDb, []byte("b"))  // table.Blocks

			it := src.NewIterator()
			defer it.Release()

			for it.Next() {
				block := new(inter.Block)
				err = rlp.DecodeBytes(it.Value(), block)
				if err != nil {
					return
				}

				info := blockInfo(evmcore.ToEvmHeader(block))
				s.set(dst, it.Key(), info)
			}

			err = it.Error()
			return

		})
}

// tablesToMoveFromGossip is a snapshot of Store.tables for migration
type tablesToMoveFromGossip struct {
	Genesis                    kvdb.KeyValueStore `table:"G"`
	ActiveValidationScore      kvdb.KeyValueStore `table:"V"`
	DirtyValidationScore       kvdb.KeyValueStore `table:"v"`
	ActiveOriginationScore     kvdb.KeyValueStore `table:"O"`
	DirtyOriginationScore      kvdb.KeyValueStore `table:"o"`
	BlockDowntime              kvdb.KeyValueStore `table:"m"`
	StakerPOIScore             kvdb.KeyValueStore `table:"s"`
	AddressPOIScore            kvdb.KeyValueStore `table:"a"`
	AddressFee                 kvdb.KeyValueStore `table:"g"`
	StakerDelegatorsFee        kvdb.KeyValueStore `table:"d"`
	AddressLastTxTime          kvdb.KeyValueStore `table:"X"`
	TotalPoiFee                kvdb.KeyValueStore `table:"U"`
	Validators                 kvdb.KeyValueStore `table:"1"`
	Stakers                    kvdb.KeyValueStore `table:"2"`
	Delegators                 kvdb.KeyValueStore `table:"3"`
	SfcConstants               kvdb.KeyValueStore `table:"4"`
	TotalSupply                kvdb.KeyValueStore `table:"5"`
	Receipts                   kvdb.KeyValueStore `table:"r"`
	DelegatorOldRewards        kvdb.KeyValueStore `table:"6"`
	StakerOldRewards           kvdb.KeyValueStore `table:"7"`
	StakerDelegatorsOldRewards kvdb.KeyValueStore `table:"8"`
	ForEvmTable                kvdb.KeyValueStore `table:"M"`
	ForEvmLogsTable            kvdb.KeyValueStore `table:"L"`
	EpochStats                 kvdb.KeyValueStore `table:"E"`
}

// engineCheckpoint is a snapshot of poset.Checkpoint for migration
type engineCheckpoint struct {
	LastDecidedFrame idx.Frame
	LastBlockN       idx.Block
	LastAtropos      hash.Event
	AppHash          common.Hash
}