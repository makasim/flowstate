package sqlitedriver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/makasim/flowstate"
)

type DataLog struct {
	db *sql.DB
}

func NewDataLog(db *sql.DB) *DataLog {
	return &DataLog{
		db: db,
	}
}

func (l *DataLog) Init(_ *flowstate.Engine) error {
	return nil
}

func (l *DataLog) Shutdown(_ context.Context) error {
	return nil
}

func (l *DataLog) Do(cmd0 flowstate.Command) error {
	if cmd, ok := cmd0.(*flowstate.StoreDataCommand); ok {
		if err := cmd.Prepare(); err != nil {
			return err
		}

		return l.Add(cmd.Data)
	}
	if cmd, ok := cmd0.(*flowstate.GetDataCommand); ok {
		if err := cmd.Prepare(); err != nil {
			return err
		}

		return l.Get(cmd.Data)
	}

	return flowstate.ErrCommandNotSupported
}

func (l *DataLog) Add(data *flowstate.Data) error {
	var nextRev int64
	res, err := l.db.Exec(`INSERT INTO flowstate_data_log(id, rev, data) VALUES(?, NULL, ?)`, data.ID, data.B)
	if err != nil {
		return fmt.Errorf("db: insert: %w", err)
	}
	nextRev, err = res.LastInsertId()
	if err != nil {
		return fmt.Errorf("db: query next rev: last insert id: %w", err)
	}
	data.Rev = nextRev
	return nil
}

func (_ *DataLog) AddTx(tx *sql.Tx, data *flowstate.Data) error {
	var nextRev int64
	res, err := tx.Exec(`INSERT INTO flowstate_data_log(id, rev, data) VALUES(?, NULL, ?)`, data.ID, data.B)
	if err != nil {
		return fmt.Errorf("db: insert: %w", err)
	}
	nextRev, err = res.LastInsertId()
	if err != nil {
		return fmt.Errorf("db: query next rev: last insert id: %w", err)
	}
	data.Rev = nextRev
	return nil
}

func (l *DataLog) Get(data *flowstate.Data) error {
	row := l.db.QueryRow(`SELECT data FROM flowstate_data_log WHERE id = ? AND rev = ?`, data.ID, data.Rev)
	if err := row.Scan(&data.B); err != nil {
		return fmt.Errorf("db: select: %w", err)
	}

	return nil
}

func (_ *DataLog) GetTx(tx *sql.Tx, data *flowstate.Data) error {
	row := tx.QueryRow(`SELECT data FROM flowstate_data_log WHERE id = ? AND rev = ?`, data.ID, data.Rev)
	if err := row.Scan(&data.B); err != nil {
		return fmt.Errorf("db: select: %w", err)
	}

	return nil
}
