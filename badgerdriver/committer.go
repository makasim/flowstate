package badgerdriver

import (
	"context"
	"errors"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Commiter{}

type Commiter struct {
	db  *badger.DB
	seq *badger.Sequence

	e flowstate.Engine
}

func NewCommiter(db *badger.DB, seq *badger.Sequence) *Commiter {
	return &Commiter{
		db:  db,
		seq: seq,
	}
}

func (d *Commiter) Do(cmd0 flowstate.Command) error {
	if _, ok := cmd0.(*flowstate.CommitStateCtxCommand); ok {
		return nil
	}

	cmd, ok := cmd0.(*flowstate.CommitCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if len(cmd.Commands) == 0 {
		return fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*flowstate.CommitCommand); ok {
			return fmt.Errorf("commit command not allowed inside another commit")
		}
		if _, ok := c.(*flowstate.ExecuteCommand); ok {
			return fmt.Errorf("execute command not allowed inside commit")
		}
	}

	cmdCtxs := make([]cmdCtx, len(cmd.Commands))
	for i := range cmd.Commands {
		cmdCtxs[i] = cmdCtx{
			cmd: cmd.Commands[i],
		}
	}

	var attempt int
	var maxAttempts = len(cmd.Commands)
	for {
		if err := d.db.Update(func(txn *badger.Txn) error {
			for i, cmd0 := range cmd.Commands {
				if cmdCtxs[i].done {
					continue
				}

				if err := d.e.Do(cmd0); err != nil {
					return fmt.Errorf("%T: do: %w", cmd0, err)
				}

				cmdCtxs[i].done = true

				cmd1, ok := cmd0.(flowstate.CommittableCommand)
				if !ok {
					continue
				}

				stateCtx := cmd1.CommittableStateCtx()
				if stateCtx.Current.ID == `` {
					return fmt.Errorf("state id empty")
				}

				cmdCtxs[i].committableStateCtx = stateCtx.CopyTo(&flowstate.StateCtx{})
			}

			for i := range cmdCtxs {
				if cmdCtxs[i].committableStateCtx == nil {
					continue
				}
				stateCtx := cmdCtxs[i].committableStateCtx

				commitedRev, err := getLatestStateRev(txn, stateCtx)
				if err != nil {
					return err
				}
				if stateCtx.Committed.Rev != commitedRev {
					conflictErr := &flowstate.ErrRevMismatch{}
					conflictErr.Add(fmt.Sprintf("%T", cmdCtxs[i].cmd), stateCtx.Current.ID, nil)
					return conflictErr
				}

				nextRev, err := d.seq.Next()
				if err != nil {
					return fmt.Errorf("get next sequence: %w", err)
				}
				stateCtx.Current.Rev = int64(nextRev)

				if err := setState(txn, stateCtx); err != nil {
					return fmt.Errorf("set state: %w", err)
				}
				if err := setLatestStateRev(txn, stateCtx); err != nil {
					return fmt.Errorf("set state: %w", err)
				}

				for lk, lv := range stateCtx.Current.Labels {
					if err := setStateLabel(txn, stateCtx.Current.ID, stateCtx.Current.Rev, lk, lv); err != nil {
						return err
					}
				}

				// TODO: build labels index
			}

			return nil
		}); errors.Is(err, badger.ErrConflict) {
			if attempt < maxAttempts {
				attempt++
				continue
			}

			return err
		} else if err != nil {
			return err
		}

		return nil
	}
}

func (d *Commiter) Init(e flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Commiter) Shutdown(_ context.Context) error {
	return nil
}

type cmdCtx struct {
	cmd                 flowstate.Command
	done                bool
	committableStateCtx *flowstate.StateCtx
}
