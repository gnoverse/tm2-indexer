package db

import (
	_ "database/sql"
	"fmt"

	_ "embed"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	//go:embed schemas.sql
	sqlSchema string
)

type DB struct {
	conn *sqlx.DB
}

func NewDB(uri string) (*DB, error) {
	var err error
	db := &DB{}

	db.conn, err = sqlx.Open("postgres", uri)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) InitTables() error {
	_, err := db.conn.Exec(sqlSchema)
	return err
}

func (db *DB) Close() error {
	return db.Close()
}

func (db *DB) InsertBlocks(blocks []*Block) error {
	tx := db.conn.MustBegin()

	sqInsertBlocks := sq.Insert("blocks").
		Columns(
			"height",
			"time",
			"hash",
			"num_txs",
			"total_txs",
			"app_version",
			"data_hash",
			"last_commit_hash",
			"validators_hash",
			"next_validators_hash",
			"consensus_hash",
			"app_hash",
			"last_results_hash",
			"proposer_address",
		).
		Suffix("ON CONFLICT DO NOTHING")

	sqInsertBlockSigns := sq.Insert("block_signatures").
		Columns("block_height", "validator_addr", "signed").
		Suffix("ON CONFLICT DO NOTHING")

	sqInsertTransactions := sq.Insert("transactions").
		Columns("height", "tx_index", "hash", "gas_fee", "gas_wanted", "memo").
		Suffix("ON CONFLICT DO NOTHING")
	_ = sqInsertTransactions

	sqInsertMessages := sq.Insert("messages").
		Columns("height", "tx_hash", "index", "route", "type", "msg_raw").
		Suffix("ON CONFLICT DO NOTHING")

	for _, b := range blocks {
		sqInsertBlocks = sqInsertBlocks.Values(
			b.Height,
			b.Time,
			b.Hash,
			b.NumTXS,
			b.TotalTXS,
			b.AppVersion,
			b.DataHash,
			b.LastCommitHash,
			b.ValidatorsHash,
			b.NextValidatorsHash,
			b.ConsensusHash,
			b.AppHash,
			b.LastResultsHash,
			b.ProposerAddress,
		)

		for _, sign := range b.Signatures {
			sqInsertBlockSigns = sqInsertBlockSigns.Values(b.Height, sign.ValidatorAddr, sign.Signed)
		}

		for _, tx := range b.Transactions {
			sqInsertTransactions = sqInsertTransactions.Values(
				tx.Height,
				tx.Index,
				tx.Hash,
				tx.GasFee,
				tx.GasWanted,
				tx.Memo,
			)
			for _, msg := range tx.Messages {
				sqInsertMessages = sqInsertMessages.Values(
					msg.Height, msg.TxHash, msg.Index, msg.Route, msg.Type, msg.MsgRaw,
				)
			}
		}
	}

	_, err := sqInsertBlocks.PlaceholderFormat(sq.Dollar).RunWith(tx).Exec()
	if err != nil {
		return fmt.Errorf("failed to insert: block, %w", err)
	}

	_, err = sqInsertBlockSigns.PlaceholderFormat(sq.Dollar).RunWith(tx).Exec()
	if err != nil {
		for _, b := range blocks {
			fmt.Printf("block[%v] - %v\n", b.Height, len(b.Signatures))
		}
		return fmt.Errorf("failed to insert: block_signatures, %w", err)
	}

	if _, args, _ := sqInsertTransactions.ToSql(); len(args) > 0 {
		_, err = sqInsertTransactions.PlaceholderFormat(sq.Dollar).RunWith(tx).Exec()
		if err != nil {
			return fmt.Errorf("failed to insert: transactions, %w", err)
		}
	}

	if _, args, _ := sqInsertMessages.ToSql(); len(args) > 0 {
		_, err = sqInsertMessages.PlaceholderFormat(sq.Dollar).RunWith(tx).Exec()
		if err != nil {
			return fmt.Errorf("failed to insert: messages, %w", err)
		}
	}

	return tx.Commit()
}

func (db *DB) InsertValidators(validators []Validator) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO validators(validator_addr, validator_name)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, val := range validators {
		if _, err = stmt.Exec(val.Addr, val.Name); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) GetMissingBlocksInSeries() ([]int64, error) {
	var heights []int64

	err := db.conn.Select(&heights, `
SELECT gs.height
FROM generate_series(1, (SELECT MAX(height) FROM blocks)) AS gs(height)
LEFT JOIN blocks b ON gs.height = b.height
WHERE b.height IS NULL
ORDER BY gs.height;
`)

	return heights, err
}

func (db *DB) GetLatestBlockHeight() (int64, error) {
	var height int64

	err := db.conn.Get(&height, `SELECT COALESCE(MAX(height), 1) FROM blocks`)

	return height, err
}
