package badgerdriver

import (
	"context"
	"reflect"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func TestCommitter_CommitNewStateOK(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}

	seq, err := db.GetSequence([]byte("seq"), 100)
	if err != nil {
		t.Fatalf("failed to get sequence: %v", err)
	}
	if _, err := seq.Next(); err != nil {
		t.Fatalf("failed to get next sequence value: %v", err)
	}

	c := NewCommiter(db, seq)
	if err := c.Init(&stubEngine{}); err != nil {
		t.Fatalf("failed to init engine: %v", err)
	}
	defer func() {
		if err := c.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown commiter: %v", err)
		}
	}()

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aStateID",
			Rev: 0,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	}
	if err := c.Do(flowstate.Commit(flowstate.CommitStateCtx(stateCtx))); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if stateCtx.Current.Rev == 0 {
		t.Fatal("expected current state revision to be set, got 0")
	}
	if stateCtx.Committed.Rev == 0 {
		t.Fatal("expected commited state revision to be set, got 0")
	}
	if stateCtx.Committed.CommittedAtUnixMilli == 0 {
		t.Fatal("expected commited state committed at unix milli to be set, got 0")
	}

	if err := db.View(func(txn *badger.Txn) error {
		commitedState, err := getState(txn, `aStateID`, stateCtx.Committed.Rev)
		if err != nil {
			t.Fatal("failed to get commited state:", err)
		}
		if !reflect.DeepEqual(stateCtx.Committed, commitedState) {
			t.Fatal("commited state does not match expected state")
		}

		commitedRev, err := getLatestStateRev(txn, commitedState)
		if err != nil {
			t.Fatal("failed to get latest state revision:", err)
		}
		if commitedRev != stateCtx.Committed.Rev {
			t.Fatal("latest state revision does not match commited state revision")
		}

		return nil
	}); err != nil {
		t.Fatalf("failed to view db: %v", err)
	}
}

func TestCommitter_CommitNewStateRevMismatch(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}

	seq, err := db.GetSequence([]byte("seq"), 100)
	if err != nil {
		t.Fatalf("failed to get sequence: %v", err)
	}
	if _, err := seq.Next(); err != nil {
		t.Fatalf("failed to get next sequence value: %v", err)
	}

	c := NewCommiter(db, seq)
	if err := c.Init(&stubEngine{}); err != nil {
		t.Fatalf("failed to init engine: %v", err)
	}
	defer func() {
		if err := c.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown commiter: %v", err)
		}
	}()

	if err := db.Update(func(txn *badger.Txn) error {
		if err := setLatestStateRev(txn, flowstate.State{ID: `aStateID`, Rev: 123}); err != nil {
			t.Fatalf("failed to set state: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("failed to update db: %v", err)
	}

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aStateID",
			Rev: 0,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
		},
	}
	if err := c.Do(flowstate.Commit(flowstate.CommitStateCtx(stateCtx))); !flowstate.IsErrRevMismatch(err) {
		t.Fatalf("expected rev mismatch error, got: %v", err)
	}

	if stateCtx.Current.Rev != 0 {
		t.Fatal("expected current state revision to be 0, got:", stateCtx.Current.Rev)
	}
	if stateCtx.Committed.Rev != 0 {
		t.Fatal("expected commited state revision to be 0, got:", stateCtx.Committed.Rev)
	}
	if stateCtx.Committed.CommittedAtUnixMilli != 0 {
		t.Fatal("expected commited state committed at unix milli to be 0, got:", stateCtx.Committed.CommittedAtUnixMilli)
	}
}

type stubEngine struct{}

func (e *stubEngine) Execute(_ *flowstate.StateCtx) error {
	panic("engine: execute: should not be called")
	return nil
}

func (e *stubEngine) Do(_ ...flowstate.Command) error {
	return nil
}

func (e *stubEngine) Shutdown(_ context.Context) error {
	panic("engine: shutdown: should not be called")
	return nil
}
