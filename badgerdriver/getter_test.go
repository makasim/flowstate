package badgerdriver

import (
	"context"
	"reflect"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func TestGetter_GetByIDOK(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}
	seq, err := getStateRevSequence(db)
	if err != nil {
		t.Fatalf("failed to get state rev sequence: %v", err)
	}

	storeTestState(t, db, seq, flowstate.State{
		ID: `aStateID1`,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})
	storeTestState(t, db, seq, flowstate.State{
		ID: `aStateID1`,
		Annotations: map[string]string{
			"annotation2": "value2",
		},
	})
	storeTestState(t, db, seq, flowstate.State{
		ID: `aStateID2`,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	f := func(cmd *flowstate.GetCommand, expState flowstate.State) {
		g := NewGetter(db)
		if err := g.Init(&stubEngine{}); err != nil {
			t.Fatalf("failed to init engine: %v", err)
		}
		defer func() {
			if err := g.Shutdown(context.Background()); err != nil {
				t.Fatalf("failed to shutdown getter: %v", err)
			}
		}()

		if err := g.Do(cmd); err != nil {
			t.Fatalf("failed to get state by ID: %v", err)
		}

		if !reflect.DeepEqual(cmd.StateCtx.Current, expState) {
			t.Fatalf("expected current state %v, got %v", expState, cmd.StateCtx.Current)
		}
		if !reflect.DeepEqual(cmd.StateCtx.Committed, expState) {
			t.Fatalf("expected committed state %v, got %v", expState, cmd.StateCtx.Current)
		}
	}

	// find latest by id
	f(flowstate.GetByID(&flowstate.StateCtx{}, "aStateID1", 0), flowstate.State{
		ID:  `aStateID1`,
		Rev: 2,
		Annotations: map[string]string{
			"annotation2": "value2",
		},
	})
	// another find latest by id
	f(flowstate.GetByID(&flowstate.StateCtx{}, "aStateID2", 0), flowstate.State{
		ID:  `aStateID2`,
		Rev: 3,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	// find by id and rev
	f(flowstate.GetByID(&flowstate.StateCtx{}, "aStateID1", 1), flowstate.State{
		ID:  `aStateID1`,
		Rev: 1,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	// another find by id and rev
	f(flowstate.GetByID(&flowstate.StateCtx{}, "aStateID2", 3), flowstate.State{
		ID:  `aStateID2`,
		Rev: 3,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})
}

func TestGetter_GetByIDError(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
	if err != nil {
		t.Fatalf("failed to open badger db: %v", err)
	}
	seq, err := getStateRevSequence(db)
	if err != nil {
		t.Fatalf("failed to get state rev sequence: %v", err)
	}

	storeTestState(t, db, seq, flowstate.State{
		ID: `aStateID1`,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	f := func(cmd *flowstate.GetCommand, errStr string) {
		g := NewGetter(db)
		if err := g.Init(&stubEngine{}); err != nil {
			t.Fatalf("failed to init engine: %v", err)
		}
		defer func() {
			if err := g.Shutdown(context.Background()); err != nil {
				t.Fatalf("failed to shutdown getter: %v", err)
			}
		}()

		err := g.Do(cmd)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if err.Error() != errStr {
			t.Fatalf("expected error %q, got %q", errStr, err.Error())
		}
		if !reflect.DeepEqual(cmd.StateCtx.Current, flowstate.State{}) {
			t.Fatalf("expected empty current state, got %v", cmd.StateCtx.Current)
		}
		if !reflect.DeepEqual(cmd.StateCtx.Current, flowstate.State{}) {
			t.Fatalf("expected empty committed state, got %v", cmd.StateCtx.Current)
		}
	}

	// state not found
	f(flowstate.GetByID(&flowstate.StateCtx{}, "aStateDoesNotExist", 0), "state not found; id=aStateDoesNotExist")

	// negative rev
	f(flowstate.GetByID(&flowstate.StateCtx{}, "aStateID1", -1), "invalid get command")

	// no such rev
	f(flowstate.GetByID(&flowstate.StateCtx{}, "aStateID1", 123), "state not found; id=aStateID1")
}
