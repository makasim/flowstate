package badgerdriver

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	statesCtx := make([]*flowstate.StateCtx, len(cmd.Commands))
	commitedStates := make([]flowstate.State, len(cmd.Commands))
	for i, cmd0 := range cmd.Commands {
		if err := d.e.Do(cmd0); err != nil {
			return fmt.Errorf("%T: do: %w", cmd0, err)
		}

		cmd1, ok := cmd0.(flowstate.CommittableCommand)
		if !ok {
			continue
		}

		stateCtx := cmd1.CommittableStateCtx()
		if stateCtx.Current.ID == `` {
			return fmt.Errorf("state id empty")
		}

		statesCtx[i] = stateCtx
	}

	var attempt int
	var maxAttempts = len(cmd.Commands)
	for {
		if err := d.db.Update(func(txn *badger.Txn) error {
			for i := range statesCtx {
				stateCtx := statesCtx[i]
				if statesCtx == nil {
					continue
				}

				commitedRev, err := getLatestStateRev(txn, stateCtx.Current.ID)
				if err != nil {
					return err
				}
				if stateCtx.Committed.Rev != commitedRev {
					conflictErr := &flowstate.ErrRevMismatch{}
					conflictErr.Add(fmt.Sprintf("%T", cmd.Commands[i]), stateCtx.Current.ID, nil)
					return conflictErr
				}

				nextRev, err := d.seq.Next()
				if err != nil {
					return fmt.Errorf("get next sequence: %w", err)
				}
				commitedState := stateCtx.Current.CopyTo(&commitedStates[i])
				commitedState.Rev = int64(nextRev)
				commitedState.CommittedAtUnixMilli = time.Now().UnixMilli()
				commitedStates[i] = commitedState

				if err := setState(txn, commitedState); err != nil {
					return fmt.Errorf("set state: %w", err)
				}
				if err := setLatestStateRev(txn, commitedState); err != nil {
					return fmt.Errorf("set latest state rev: %w", err)
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

		for i := range commitedStates {
			if statesCtx[i] == nil {
				continue
			}

			stateCtx := statesCtx[i]
			commitedStates[i].CopyTo(&stateCtx.Current)
			commitedStates[i].CopyTo(&stateCtx.Committed)
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
	cmd           flowstate.Command
	done          bool
	stateCtx      *flowstate.StateCtx
	commitedState flowstate.State
}
