package badgerdriver

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Getter{}

type Getter struct {
	db *badger.DB
}

func NewGetter(db *badger.DB) *Getter {
	return &Getter{
		db: db,
	}
}

func (d *Getter) Do(cmd0 flowstate.Command) error {
	switch cmd := cmd0.(type) {
	case *flowstate.GetCommand:
		return d.doGet(cmd)
	case *flowstate.GetManyCommand:
		return d.doGetMany(cmd)
	default:
		return flowstate.ErrCommandNotSupported
	}
}

func (d *Getter) doGet(cmd *flowstate.GetCommand) error {
	if len(cmd.Labels) > 0 {
		return d.db.View(func(txn *badger.Txn) error {
			it := newAndLabelIterator(txn, cmd.Labels, cmd.Rev)
			defer it.Close()

			if !it.Valid() {
				return fmt.Errorf("%w; labels=%v", flowstate.ErrNotFound, cmd.Labels)
			}

			state := it.Current()
			state.CopyToCtx(cmd.StateCtx)
			return nil
		})
	} else if cmd.ID != "" && cmd.Rev >= 0 {
		return d.db.View(func(txn *badger.Txn) error {
			rev := cmd.Rev
			if rev == 0 {
				var err error
				rev, err = getLatestStateRev(txn, cmd.ID)
				if err != nil {
					return err
				}
			}

			state, err := getState(txn, cmd.ID, rev)
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("%w; id=%s", flowstate.ErrNotFound, cmd.ID)
			} else if err != nil {
				return err
			}

			state.CopyToCtx(cmd.StateCtx)
			return nil
		})
	} else {
		return fmt.Errorf("invalid get command")
	}
}

func (d *Getter) doGetMany(cmd *flowstate.GetManyCommand) error {
	cmd.Prepare()

	return fmt.Errorf("get many command not implemented yet")
}

func (d *Getter) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Getter) Shutdown(_ context.Context) error {
	return nil
}
