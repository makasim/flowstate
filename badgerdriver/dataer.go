package badgerdriver

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

type Dataer struct {
	db  *badger.DB
	seq *badger.Sequence
}

func NewDataer(db *badger.DB, seq *badger.Sequence) *Dataer {
	return &Dataer{
		db:  db,
		seq: seq,
	}
}

func (d *Dataer) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Dataer) Shutdown(_ context.Context) error {
	return nil
}

func (d *Dataer) Do(cmd0 flowstate.Command) error {
	if cmd, ok := cmd0.(*flowstate.StoreDataCommand); ok {
		if err := cmd.Prepare(); err != nil {
			return err
		}

		nextRev, err := d.seq.Next()
		if err != nil {
			return fmt.Errorf("get next sequence: %w", err)
		}

		data := cmd.Data
		data.Rev = int64(nextRev)

		return d.db.Update(func(txn *badger.Txn) error {
			return setData(txn, data)
		})
	}
	if cmd, ok := cmd0.(*flowstate.GetDataCommand); ok {
		if err := cmd.Prepare(); err != nil {
			return err
		}

		return d.db.View(func(txn *badger.Txn) error {
			return getData(txn, cmd.Data)
		})
	}

	return flowstate.ErrCommandNotSupported
}
