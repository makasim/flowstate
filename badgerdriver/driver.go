package badgerdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

type Driver struct {
	*flowstate.FlowRegistry

	db          *badger.DB
	stateRevSeq *badger.Sequence
	dataRevSeq  *badger.Sequence

	l *slog.Logger
}

func New(db *badger.DB) (*Driver, error) {
	stateRevSeq, err := getStateRevSequence(db)
	if err != nil {
		return nil, fmt.Errorf("db: get state rev seq: %w", err)
	}
	dataRevSeq, err := getDataRevSequence(db)
	if err != nil {
		return nil, fmt.Errorf("db: get data rev seq: %w", err)
	}

	return &Driver{
		db:           db,
		stateRevSeq:  stateRevSeq,
		dataRevSeq:   dataRevSeq,
		FlowRegistry: &flowstate.FlowRegistry{},

		l: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
	}, nil
}

func (d *Driver) Shutdown(_ context.Context) error {
	var res error

	if err := d.stateRevSeq.Release(); err != nil {
		res = errors.Join(res, fmt.Errorf("release state rev seq: %w", err))
	}
	if err := d.dataRevSeq.Release(); err != nil {
		res = errors.Join(res, fmt.Errorf("release data rev seq: %w", err))
	}

	return res
}

func (d *Driver) GetData(cmd *flowstate.GetDataCommand) error {
	return d.db.View(func(txn *badger.Txn) error {
		return getData(txn, cmd.Data)
	})
}

func (d *Driver) StoreData(cmd *flowstate.StoreDataCommand) error {
	nextRev, err := d.dataRevSeq.Next()
	if err != nil {
		return fmt.Errorf("get next sequence: %w", err)
	}

	data := cmd.Data
	data.Rev = int64(nextRev)

	return d.db.Update(func(txn *badger.Txn) error {
		return setData(txn, data)
	})
}

func (d *Driver) GetStateByID(cmd *flowstate.GetStateByIDCommand) error {
	if cmd.Rev < 0 {
		return fmt.Errorf("invalid revision: %d; must be >= 0", cmd.Rev)
	}

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
}

func (d *Driver) GetStateByLabels(cmd *flowstate.GetStateByLabelsCommand) error {
	return d.db.View(func(txn *badger.Txn) error {
		it := newLabelsIterator(txn, cmd.Labels, 0, true)
		defer it.Close()

		if !it.Valid() {
			return fmt.Errorf("%w; labels=%v", flowstate.ErrNotFound, cmd.Labels)
		}

		state := it.Current()
		state.CopyToCtx(cmd.StateCtx)
		return nil
	})
}

func (d *Driver) GetStates(cmd *flowstate.GetStatesCommand) (*flowstate.GetStatesResult, error) {
	res := &flowstate.GetStatesResult{}
	if err := d.db.View(func(txn *badger.Txn) error {
		sinceRev := cmd.SinceRev
		if !cmd.SinceTime.IsZero() {
			caIt := newCommittedAtIterator(txn, cmd.SinceTime, false)
			defer caIt.Close()

			if !caIt.Valid() {
				return nil
			}

			sinceRev = caIt.Current().Rev
		} else if sinceRev == -1 {
			latestIt := newOrLabelsIterator(txn, cmd.Labels, 0, true)
			defer latestIt.Close()

			if !latestIt.Valid() {
				return nil
			}

			sinceRev = latestIt.Current().Rev
		} else if sinceRev > 0 {
			sinceRev = sinceRev + 1
		}

		it := newOrLabelsIterator(txn, cmd.Labels, sinceRev, false)
		defer it.Close()

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
		return nil
	}); err != nil {
		return nil, fmt.Errorf("get states: %w", err)
	}

	return res, nil
}

func (d *Driver) Delay(cmd *flowstate.DelayCommand) error {
	return fmt.Errorf("not implemented")
}

func (d *Driver) GetDelayedStates(cmd *flowstate.GetDelayedStatesCommand) (*flowstate.GetDelayedStatesResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *Driver) Commit(cmd *flowstate.CommitCommand, e flowstate.Engine) error {
	var attempt int
	var maxAttempts = len(cmd.Commands)
	for {
		if err := d.db.Update(func(txn *badger.Txn) error {
			for i, subCmd0 := range cmd.Commands {
				if _, ok := subCmd0.(*flowstate.CommitStateCtxCommand); !ok {
					if err := e.Do(subCmd0); err != nil {
						return fmt.Errorf("%T: do: %w", subCmd0, err)
					}
				}

				subCmd, ok := subCmd0.(flowstate.CommittableCommand)
				if !ok {
					continue
				}

				stateCtx := subCmd.CommittableStateCtx()
				if stateCtx.Current.ID == `` {
					return fmt.Errorf("state id empty")
				}

				commitedRev, err := getLatestRevIndex(txn, stateCtx.Current.ID)
				if err != nil {
					return err
				}
				if stateCtx.Committed.Rev != commitedRev {
					conflictErr := &flowstate.ErrRevMismatch{}
					conflictErr.Add(fmt.Sprintf("%T", cmd.Commands[i]), stateCtx.Current.ID, nil)
					return conflictErr
				}

				nextRev, err := d.stateRevSeq.Next()
				if err != nil {
					return fmt.Errorf("get next sequence: %w", err)
				}
				commitedState := stateCtx.Current.CopyTo(&flowstate.State{})
				commitedState.Rev = int64(nextRev)
				commitedState.CommittedAtUnixMilli = time.Now().UnixMilli()

				if err := setState(txn, commitedState); err != nil {
					return fmt.Errorf("set state: %w", err)
				}
				if err := setLatestRevIndex(txn, commitedState); err != nil {
					return fmt.Errorf("set latest rev index: %w", err)
				}
				if err := setLabelsIndex(txn, commitedState); err != nil {
					return fmt.Errorf("set labels index: %w", err)
				}
				if err := setRevIndex(txn, commitedState); err != nil {
					return fmt.Errorf("set rev index: %w", err)
				}
				if err := setCommittedAtIndex(txn, commitedState); err != nil {
					return fmt.Errorf("set committed at index: %w", err)
				}

				commitedState.CopyToCtx(stateCtx)
				stateCtx.Transitions = stateCtx.Transitions[:0]
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

func (d *Driver) GetFlow(cmd *flowstate.GetFlowCommand) error {
	return d.FlowRegistry.Do(cmd)
}

func getStateRevSequence(db *badger.DB) (*badger.Sequence, error) {
	seq, err := db.GetSequence([]byte("flowstate.rev.state"), 10000)
	if err != nil {
		return nil, err
	}
	// make sure we never get rev=0
	if _, err := seq.Next(); err != nil {
		return nil, err
	}

	return seq, nil
}

func getDataRevSequence(db *badger.DB) (*badger.Sequence, error) {
	seq, err := db.GetSequence([]byte("flowstate.rev.data"), 10000)
	if err != nil {
		return nil, err
	}
	// make sure we never get rev=0
	if _, err := seq.Next(); err != nil {
		return nil, err
	}

	return seq, nil
}
