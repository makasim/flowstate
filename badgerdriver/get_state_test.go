package badgerdriver

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func TestGetStateByIDOK(t *testing.T) {
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

	storeTestState(t, d, flowstate.State{
		ID: `aStateID1`,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})
	storeTestState(t, d, flowstate.State{
		ID: `aStateID1`,
		Annotations: map[string]string{
			"annotation2": "value2",
		},
	})
	storeTestState(t, d, flowstate.State{
		ID: `aStateID2`,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	f := func(cmd *flowstate.GetStateByIDCommand, expState flowstate.State) {
		if err := d.GetStateByID(cmd); err != nil {
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
	f(flowstate.GetStateByID(&flowstate.StateCtx{}, "aStateID1", 0), flowstate.State{
		ID:  `aStateID1`,
		Rev: 2,
		Annotations: map[string]string{
			"annotation2": "value2",
		},
	})
	// another find latest by id
	f(flowstate.GetStateByID(&flowstate.StateCtx{}, "aStateID2", 0), flowstate.State{
		ID:  `aStateID2`,
		Rev: 3,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	// find by id and rev
	f(flowstate.GetStateByID(&flowstate.StateCtx{}, "aStateID1", 1), flowstate.State{
		ID:  `aStateID1`,
		Rev: 1,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	// another find by id and rev
	f(flowstate.GetStateByID(&flowstate.StateCtx{}, "aStateID2", 3), flowstate.State{
		ID:  `aStateID2`,
		Rev: 3,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})
}

func TestGetByIDError(t *testing.T) {
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

	storeTestState(t, d, flowstate.State{
		ID: `aStateID1`,
		Annotations: map[string]string{
			"annotation1": "value1",
		},
	})

	f := func(cmd *flowstate.GetStateByIDCommand, errStr string) {
		t.Helper()

		err := d.GetStateByID(cmd)
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
	f(flowstate.GetStateByID(&flowstate.StateCtx{}, "aStateDoesNotExist", 0), "state not found; id=aStateDoesNotExist")

	// negative rev
	f(flowstate.GetStateByID(&flowstate.StateCtx{}, "aStateID1", -1), "invalid revision: -1; must be >= 0")

	// no such rev
	f(flowstate.GetStateByID(&flowstate.StateCtx{}, "aStateID1", 123), "state not found; id=aStateID1")
}

// TODO: there is no way to introduce a required delay in badger driver, so this test is disabled
// Earlier it worked because Commit accepted flowstate.Engine and its methods have been called inside transaction.
//func TestGetStatesPreserveOrder(t *testing.T) {
//	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
//	if err != nil {
//		t.Fatalf("failed to open badger db: %v", err)
//	}
//	defer func() {
//		if err := db.Close(); err != nil {
//			t.Fatalf("failed to close db: %v", err)
//		}
//	}()
//
//	d, err := New(db)
//	if err != nil {
//		t.Fatalf("failed to create driver: %v", err)
//	}
//	defer func() {
//		if err := d.Shutdown(context.Background()); err != nil {
//			t.Fatalf("failed to shutdown commiter: %v", err)
//		}
//	}()
//
//	// past
//	if err := d.Commit(flowstate.Commit(flowstate.CommitStateCtx(&flowstate.StateCtx{
//		Current: flowstate.State{ID: `1`},
//	}))); err != nil {
//		t.Fatalf("failed to commit state: %v", err)
//	}
//
//	var wg sync.WaitGroup
//
//	// start first commit later
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//
//		if err := d.Commit(flowstate.Commit(flowstate.Pause(&flowstate.StateCtx{
//			Current: flowstate.State{ID: `2`},
//		})), &doDelayEngine{dur: time.Millisecond * 600}); err != nil {
//			t.Fatalf("failed to commit state: %v", err)
//		}
//	}()
//
//	time.Sleep(time.Millisecond * 200)
//
//	// started later but committed first
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//
//		if err := d.Commit(flowstate.Commit(flowstate.Pause(&flowstate.StateCtx{
//			Current: flowstate.State{ID: `3`},
//		}))); err != nil {
//			t.Fatalf("failed to commit state: %v", err)
//		}
//	}()
//
//	// read returns only past commited state
//
//	time.Sleep(time.Millisecond * 200)
//	res, err := d.GetStates(flowstate.GetStatesByLabels(nil).WithLimit(10))
//	if err != nil {
//		t.Fatalf("failed to get states: %v", err)
//	}
//
//	expStates := []flowstate.State{
//		{ID: `1`, Rev: 1},
//	}
//	assertEqualStates(t, expStates, res.States)
//
//	wg.Wait()
//
//	res, err = d.GetStates(flowstate.GetStatesByLabels(nil).WithLimit(10))
//	if err != nil {
//		t.Fatalf("failed to get states: %v", err)
//	}
//
//	expStates = []flowstate.State{
//		{ID: `1`, Rev: 1},
//		{ID: `2`, Rev: 2},
//		{ID: `3`, Rev: 3},
//	}
//
//	assertEqualStates(t, expStates, res.States)
//}

func assertEqualStates(t *testing.T, exp, act []flowstate.State) {
	t.Helper()

	for i := range act {
		s := act[i]
		s.CommittedAtUnixMilli = 0
		act[i] = s
	}

	if len(exp) != len(act) {
		t.Fatalf("expected states %v, got %v", exp, act)
	}
	if !reflect.DeepEqual(exp, act) {
		t.Fatalf("expected states %v, got %v", exp, act)
	}
}

type doDelayEngine struct {
	dur time.Duration
}

func (e *doDelayEngine) Execute(_ *flowstate.StateCtx) error {
	panic("engine: execute: should not be called")
	return nil
}

func (e *doDelayEngine) Do(_ ...flowstate.Command) error {
	time.Sleep(e.dur)
	return nil
}

func (e *doDelayEngine) Shutdown(_ context.Context) error {
	panic("engine: shutdown: should not be called")
	return nil
}
