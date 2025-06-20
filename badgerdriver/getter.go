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
			reverse := false
			if cmd.Rev == 0 {
				reverse = true
			}

			it := newLabelsIterator(txn, cmd.Labels, cmd.Rev, reverse)
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
				rev, err = getLatestRevIndex(txn, cmd.ID)
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

	return d.db.View(func(txn *badger.Txn) error {
		sinceRev := cmd.SinceRev
		if !cmd.SinceTime.IsZero() {
			caIt := newCommittedAtIterator(txn, cmd.SinceTime, false)
			defer caIt.Close()

			if !caIt.Valid() {
				cmd.SetResult(&flowstate.GetManyResult{})
				return nil
			}

			sinceRev = caIt.Current().Rev - 1
		}

		if sinceRev == -1 {
			latestIt := newOrLabelsIterator(txn, cmd.Labels, 0, true)
			defer latestIt.Close()

			if !latestIt.Valid() {
				cmd.SetResult(&flowstate.GetManyResult{})
				return nil
			}

			sinceRev = latestIt.Current().Rev - 1
		}

		it := newOrLabelsIterator(txn, cmd.Labels, sinceRev, false)
		defer it.Close()

		res := &flowstate.GetManyResult{}
		for ; it.Valid(); it.Next() {
			if len(res.States) > cmd.Limit {
				res.More = true
				break
			}

			state := it.Current()
			if cmd.LatestOnly {
				latestRev, _ := getLatestRevIndex(txn, state.ID)
				if state.Rev < latestRev {
					continue
				}
			}

			res.States = append(res.States, state)
		}
		cmd.SetResult(res)
		return nil
	})
}

func (d *Getter) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Getter) Shutdown(_ context.Context) error {
	return nil
}
