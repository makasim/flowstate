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

var _ flowstate.Driver = &Driver{}

type Driver struct {
	*flowstate.FlowRegistry

	db               *badger.DB
	dataRevSeq       *badger.Sequence
	delayedOffsetSeq *sequenceWithCommit
	stateRevSeq      *sequenceWithCommit

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
		db:         db,
		dataRevSeq: dataRevSeq,
		stateRevSeq: &sequenceWithCommit{

			seq:     stateRevSeq,
			ongoing: make(map[int64]struct{}),
		},
		delayedOffsetSeq: &sequenceWithCommit{

			seq:     delayedOffsetSeq,
			ongoing: make(map[int64]struct{}),
		},
		FlowRegistry: &flowstate.FlowRegistry{},

		l: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
	}, nil
}

func (d *Driver) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Driver) Shutdown(_ context.Context) error {
	var res error

	if err := d.stateRevSeq.seq.Release(); err != nil {
		res = errors.Join(res, fmt.Errorf("release state rev seq: %w", err))
	}
	if err := d.dataRevSeq.Release(); err != nil {
		res = errors.Join(res, fmt.Errorf("release data rev seq: %w", err))
	}
	if err := d.delayedOffsetSeq.seq.Release(); err != nil {
		res = errors.Join(res, fmt.Errorf("release delayed offset seq: %w", err))
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
		untilRev := d.stateRevSeq.maxViewable()

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
		untilRev := d.stateRevSeq.maxViewable()

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
	getOffset, commitOffset := d.delayedOffsetSeq.nextWithCommit()
	defer commitOffset()

	return d.db.Update(func(txn *badger.Txn) error {
		nextOffset, err := getOffset()
		if err != nil {
			return fmt.Errorf("get next sequence: %w", err)
		}

		delayedState := flowstate.DelayedState{
			State:     cmd.DelayingState,
			ExecuteAt: cmd.ExecuteAt,
			Offset:    nextOffset,
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
		untilOffset := d.delayedOffsetSeq.maxViewable()

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
			delayedState := flowstate.DelayedState{}
			if err := it.Item().Value(func(delayedStateKey []byte) error {
				item, err := txn.Get(delayedStateKey)
				if err != nil {
					return fmt.Errorf("get delayed state: %w", err)
				}

				return item.Value(func(d []byte) error {
					return flowstate.UnmarshalDelayedState(d, &delayedState)
				})
			}); err != nil {
				return err
			}

			if delayedState.ExecuteAt.Unix() <= cmd.Since.Unix() {
				continue
			}
			if delayedState.ExecuteAt.Unix() > cmd.Until.Unix() {
				break
			}
			if delayedState.Offset > untilOffset {
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

func (d *Driver) Commit(cmd *flowstate.CommitCommand) error {
	var attempt int
	var maxAttempts = len(cmd.Commands)

	getRev, commitRevs := d.stateRevSeq.nextWithCommit()
	defer commitRevs()

	for {

		if err := d.db.Update(func(txn *badger.Txn) error {
			for i, subCmd0 := range cmd.Commands {
				nextRev, err := getRev()
				if err != nil {
					return fmt.Errorf("get next sequence: %w", err)
				}

				if err := flowstate.DoCommitSubCommand(d, subCmd0); err != nil {
					return fmt.Errorf("%T: do: %w", subCmd0, err)
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

type sequenceWithCommit struct {
	seq          *badger.Sequence
	mux          sync.Mutex
	ongoing      map[int64]struct{}
	maxOngoing   int64
	maxCommitted int64
}

func (seq *sequenceWithCommit) nextWithCommit() (func() (int64, error), func()) {
	var txnCommitting []int64

	return func() (int64, error) {
			next0, err := seq.seq.Next()
			if err != nil {
				return 0, err
			}
			if next0 >= math.MaxInt64 {
				panic("FATAL: sequence overflow, int64 max %d value has been reached")
			}

			next := int64(next0)
			txnCommitting = append(txnCommitting, next)

			seq.mux.Lock()
			seq.ongoing[next] = struct{}{}
			seq.maxOngoing = next
			seq.mux.Unlock()

			return next, nil
		}, func() {
			if len(txnCommitting) == 0 {
				return
			}

			seq.mux.Lock()
			defer seq.mux.Unlock()

			for _, rev := range txnCommitting {
				delete(seq.ongoing, rev)
			}

			minRev := seq.maxOngoing
			for rev := range seq.ongoing {
				minRev = min(minRev, rev)
			}
			seq.maxCommitted = minRev

			txnCommitting = txnCommitting[:0]
		}
}

func (seq *sequenceWithCommit) maxViewable() int64 {
	seq.mux.Lock()
	defer seq.mux.Unlock()

	return seq.maxCommitted
}
