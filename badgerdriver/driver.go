package badgerdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

type Driver struct {
	*flowstate.FlowRegistry

	db         *badger.DB
	dataRevSeq *badger.Sequence

	delayedOffsetSeq *badger.Sequence

	stateRevMux sync.Mutex
	stateRevSeq *badger.Sequence
	stateRevs   map[int64]struct{}
	stateMaxRev int64
	stateMinRev int64

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
	delayedOffsetSeq, err := getDelayedOffsetSequence(db)
	if err != nil {
		return nil, fmt.Errorf("db: get delayed offset seq: %w", err)
	}

	return &Driver{
		db:               db,
		stateRevSeq:      stateRevSeq,
		stateRevs:        make(map[int64]struct{}),
		stateRevMux:      sync.Mutex{},
		dataRevSeq:       dataRevSeq,
		delayedOffsetSeq: delayedOffsetSeq,
		FlowRegistry:     &flowstate.FlowRegistry{},

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
		d.stateRevMux.Lock()
		untilRev := d.stateMinRev
		d.stateRevMux.Unlock()

		it := newLabelsIterator(txn, cmd.Labels, untilRev, true)
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
		d.stateRevMux.Lock()
		untilRev := d.stateMinRev
		d.stateRevMux.Unlock()

		sinceRev := cmd.SinceRev
		if !cmd.SinceTime.IsZero() {
			caIt := newCommittedAtIterator(txn, cmd.SinceTime, false)
			defer caIt.Close()

			if !caIt.Valid() {
				return nil
			}

			sinceRev = caIt.Current().Rev
		} else if sinceRev == -1 {
			latestIt := newOrLabelsIterator(txn, cmd.Labels, untilRev, true)
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

			if state.Rev > untilRev {
				break
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
	nextOffset, err := d.delayedOffsetSeq.Next()
	if err != nil {
		return fmt.Errorf("get next sequence: %w", err)
	}

	return d.db.Update(func(txn *badger.Txn) error {
		delayedState := flowstate.DelayedState{
			State:     cmd.DelayStateCtx.Current,
			ExecuteAt: flowstate.DelayedUntil(cmd.DelayStateCtx.Current),
			Offset:    int64(nextOffset),
		}

		if err := setDelayedState(txn, delayedState); err != nil {
			return fmt.Errorf("set delayed state: %w", err)
		}
		if err := setDelayedOffsetIndex(txn, delayedState); err != nil {
			return fmt.Errorf("set delayed offset index: %w", err)
		}

		return nil
	})
}

func (d *Driver) GetDelayedStates(cmd *flowstate.GetDelayedStatesCommand) (*flowstate.GetDelayedStatesResult, error) {
	res := &flowstate.GetDelayedStatesResult{}

	if err := d.db.View(func(txn *badger.Txn) error {
		prefix := delayedStatePrefix(cmd.Since)
		seekPrefix := delayedStateKey(cmd.Since.Unix(), cmd.Offset)
		if cmd.Offset > 0 || cmd.Since.IsZero() {
			prefix = delayedOffsetPrefix()
			seekPrefix = delayedOffsetKey(cmd.Offset)
		}

		it := &badgerIterator{
			prefix: prefix,
			Iterator: txn.NewIterator(badger.IteratorOptions{
				PrefetchSize: 100,
				Prefix:       prefix,
				Reverse:      false,
			}),
		}
		defer it.Close()

		it.Seek(seekPrefix)

		for ; it.Valid(); it.Next() {
			delayedStateKey, err := it.Item().ValueCopy(nil)
			if err != nil {
				return fmt.Errorf("get delayed state execute at: %w", err)
			}
			delayedState := flowstate.DelayedState{}
			if err := getGOB(txn, delayedStateKey, &delayedState); err != nil {
				return fmt.Errorf("get delayed state: %w", err)
			}
			if delayedState.ExecuteAt.Unix() <= cmd.Since.Unix() {
				continue
			}
			if delayedState.ExecuteAt.Unix() > cmd.Until.Unix() {
				break
			}
			if len(res.States) >= cmd.Limit {
				res.More = true
				break
			}

			res.States = append(res.States, delayedState)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("get delayed states: %w", err)
	}

	return res, nil
}

func (d *Driver) Commit(cmd *flowstate.CommitCommand, e flowstate.Engine) error {
	var attempt int
	var maxAttempts = len(cmd.Commands)

	getNextRev, commitRevs := d.nextStateRevWithCommit()
	defer commitRevs()

	for {

		if err := d.db.Update(func(txn *badger.Txn) error {
			for i, subCmd0 := range cmd.Commands {
				nextRev, err := getNextRev()
				if err != nil {
					return fmt.Errorf("get next sequence: %w", err)
				}

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

				commitedState := stateCtx.Current.CopyTo(&flowstate.State{})
				commitedState.Rev = nextRev
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
			commitRevs()

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

func (d *Driver) nextStateRevWithCommit() (func() (int64, error), func()) {
	var revs []int64

	return func() (int64, error) {
			next0, err := d.stateRevSeq.Next()
			if err != nil {
				return 0, err
			}
			if next0 >= math.MaxInt64 {
				panic("FATAL: sequence overflow, int64 max %d value has been reached")
			}

			next := int64(next0)
			revs = append(revs, next)

			d.stateRevMux.Lock()
			d.stateMaxRev = next
			d.stateRevs[next] = struct{}{}
			d.stateRevMux.Unlock()

			return next, nil
		}, func() {
			if len(revs) == 0 {
				return
			}

			d.stateRevMux.Lock()

			for _, rev := range revs {
				delete(d.stateRevs, rev)
			}

			minRev := d.stateMaxRev
			for rev := range d.stateRevs {
				minRev = min(minRev, rev)
			}
			revs = revs[:0]
			d.stateMinRev = minRev

			d.stateRevMux.Unlock()
		}
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

func getDelayedOffsetSequence(db *badger.DB) (*badger.Sequence, error) {
	seq, err := db.GetSequence([]byte("flowstate.offset.delayed_state"), 10000)
	if err != nil {
		return nil, err
	}
	// make sure we never get offset=0
	if _, err := seq.Next(); err != nil {
		return nil, err
	}

	return seq, nil
}
