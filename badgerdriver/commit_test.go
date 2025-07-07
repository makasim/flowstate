package badgerdriver

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func TestCommitOK(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	d, err := New(db)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer func() {
		if err := d.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown commiter: %v", err)
		}
	}()

	f := func(stateCtx *flowstate.StateCtx) {
		stateID := stateCtx.Current.ID
		prevRev := stateCtx.Committed.Rev
		prevCommitedAtUnixMilli := stateCtx.Committed.CommittedAtUnixMilli

		if err := d.Commit(flowstate.Commit(flowstate.CommitStateCtx(stateCtx))); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		if stateCtx.Current.Rev <= prevRev {
			t.Fatalf("expected current state revision to be greater than previous state revision %d, got: %d", prevRev, stateCtx.Current.Rev)
		}
		if stateCtx.Committed.Rev <= prevRev {
			t.Fatalf("expected commited state revision to be greater than previous state revision %d, got: %d", prevRev, stateCtx.Committed.Rev)
		}
		if stateCtx.Committed.CommittedAtUnixMilli <= prevCommitedAtUnixMilli {
			t.Fatalf("expected commited state committed at unix milli to be greater than previous value %d, got: %d", prevCommitedAtUnixMilli, stateCtx.Committed.CommittedAtUnixMilli)
		}

		if err := db.View(func(txn *badger.Txn) error {
			storedState, err := getState(txn, stateID, stateCtx.Committed.Rev)
			if err != nil {
				t.Fatal("failed to get commited state:", err)
			}
			if !reflect.DeepEqual(stateCtx.Committed, storedState) {
				t.Fatal("commited state does not match expected state")
			}

			storedRev, err := getLatestRevIndex(txn, storedState.ID)
			if err != nil {
				t.Fatal("failed to get latest state revision:", err)
			}
			if storedRev != stateCtx.Committed.Rev {
				t.Fatal("latest state revision does not match commited state revision")
			}

			return nil
		}); err != nil {
			t.Fatalf("failed to view db: %v", err)
		}
	}

	// state created
	f(&flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aStateID0",
			Rev: 0,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	})

	//  state updated
	storedState := storeTestState(t, d, flowstate.State{
		ID:                   `aStateID1`,
		CommittedAtUnixMilli: 123456789,
	})
	f(&flowstate.StateCtx{
		Committed: storedState,
		Current: flowstate.State{
			ID:  storedState.ID,
			Rev: storedState.Rev,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	})
}

func TestCommitRevMismatch(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	d, err := New(db)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer func() {
		if err := d.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown commiter: %v", err)
		}
	}()

	f := func(stateCtx *flowstate.StateCtx) {
		expStateCtx := stateCtx.CopyTo(&flowstate.StateCtx{})

		if err := db.Update(func(txn *badger.Txn) error {
			if err := setLatestRevIndex(txn, flowstate.State{ID: `aStateID`, Rev: 123}); err != nil {
				t.Fatalf("failed to set state: %v", err)
			}
			return nil
		}); err != nil {
			t.Fatalf("failed to update db: %v", err)
		}

		if err := d.Commit(flowstate.Commit(flowstate.CommitStateCtx(stateCtx))); !flowstate.IsErrRevMismatch(err) {
			t.Fatalf("expected rev mismatch error, got: %v", err)
		}

		if !reflect.DeepEqual(stateCtx, expStateCtx) {
			t.Fatal("stateCtx should not be modified on rev mismatch error")
		}
	}

	// no stored state with rev 123
	f(&flowstate.StateCtx{
		Committed: flowstate.State{
			ID:  "aStateID0",
			Rev: 123,
		},
		Current: flowstate.State{
			ID:  "aStateID0",
			Rev: 123,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	})

	// stored state has rev different than commited 123
	storedState1 := storeTestState(t, d, flowstate.State{
		ID:                   `aStateID1`,
		CommittedAtUnixMilli: 123456789,
	})
	f(&flowstate.StateCtx{
		Committed: flowstate.State{
			ID:  storedState1.ID,
			Rev: storedState1.Rev + 1,
		},
		Current: flowstate.State{
			ID:  storedState1.ID,
			Rev: storedState1.Rev + 1,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	})

	// create new but there is already stored state
	storedState2 := storeTestState(t, d, flowstate.State{
		ID:                   `aStateID2`,
		CommittedAtUnixMilli: 123456789,
	})
	f(&flowstate.StateCtx{
		Current: flowstate.State{
			ID:  storedState2.ID,
			Rev: 0,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	})
}

func TestCommitSeveralStatesOK(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()

	d, err := New(db)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer func() {
		if err := d.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown commiter: %v", err)
		}
	}()

	state1 := storeTestState(t, d, flowstate.State{
		ID:                   `aStateID1`,
		CommittedAtUnixMilli: 123456789,
	})
	stateCtx1 := &flowstate.StateCtx{
		Committed: state1,
		Current:   state1,
	}

	state2 := storeTestState(t, d, flowstate.State{
		ID:                   `aStateID2`,
		CommittedAtUnixMilli: 123456789,
	})
	stateCtx2 := &flowstate.StateCtx{
		Committed: state2,
		Current:   state2,
	}

	stateCtx3 := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  `aStateID3`,
			Rev: 0,
		},
	}

	rev1 := state1.Rev
	rev2 := state2.Rev

	if err := d.Commit(flowstate.Commit(
		flowstate.CommitStateCtx(stateCtx1),
		flowstate.CommitStateCtx(stateCtx2),
		flowstate.CommitStateCtx(stateCtx3),
	)); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if stateCtx1.Current.Rev <= rev1 {
		t.Fatalf("expected current state1 revision to be greater than revision %d, got: %d", rev1, stateCtx1.Current.Rev)
	}
	if stateCtx2.Current.Rev <= rev2 {
		t.Fatalf("expected current state2 revision to be greater than revision %d, got: %d", rev2, stateCtx2.Current.Rev)
	}
	if stateCtx3.Current.Rev <= rev2 {
		t.Fatalf("expected current state3 revision to be greater than revision %d, got: %d", rev2, stateCtx3.Current.Rev)
	}
}

func TestCommitConcurrently(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}
	defer db.Close()

	d, err := New(db)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer func() {
		if err := d.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown commiter: %v", err)
		}
	}()

	storeTestState(t, d, flowstate.State{
		ID: `aStateID0`,
	})

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			stateCtx := &flowstate.StateCtx{}
			for j := 0; j < 50; j++ {
				if err := db.View(func(txn *badger.Txn) error {
					rev, err := getLatestRevIndex(txn, `aStateID0`)
					if err != nil {
						t.Fatalf("failed to get latest state revision: %v", err)
					}
					state, err := getState(txn, `aStateID0`, rev)
					if err != nil {
						t.Fatalf("failed to get state: %v", err)
					}

					state.CopyToCtx(stateCtx)
					return nil
				}); err != nil {
					t.Fatalf("failed to view db: %v", err)
				}

				err := d.Commit(flowstate.Commit(flowstate.CommitStateCtx(stateCtx)))
				if !flowstate.IsErrRevMismatch(err) && err != nil {
					t.Fatalf("failed to commit state: %v", err)
				}
			}
		}()
	}

	wg.Wait()
}

func storeTestState(t *testing.T, d *Driver, state flowstate.State) flowstate.State {
	rev, err := d.stateRevSeq.seq.Next()
	if err != nil {
		t.Fatalf("failed to get next sequence value: %v", err)
	}

	state.Rev = int64(rev)

	if err := d.db.Update(func(txn *badger.Txn) error {
		if err := setState(txn, state); err != nil {
			t.Fatalf("failed to set state: %v", err)
		}
		if err := setLatestRevIndex(txn, state); err != nil {
			t.Fatalf("failed to set latest rev index: %v", err)
		}
		if err := setLabelsIndex(txn, state); err != nil {
			t.Fatalf("failed to set labels index: %v", err)
		}
		if err := setRevIndex(txn, state); err != nil {
			t.Fatalf("failed to set rev index: %v", err)
		}
		if err := setCommittedAtIndex(txn, state); err != nil {
			t.Fatalf("failed to set committed at index: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("failed to update db: %v", err)
	}

	return state
}
