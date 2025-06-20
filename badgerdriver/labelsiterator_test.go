package badgerdriver

import (
	"math"
	"reflect"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func TestLabelsIterator(t *testing.T) {
	f := func(storedStates []flowstate.State, labels map[string]string, sinceRev int64, reverse bool, expRes []flowstate.State) {
		t.Helper()
		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
		if err != nil {
			t.Fatalf("failed to open badger db: %v", err)
		}
		seq, err := getStateRevSequence(db)
		if err != nil {
			t.Fatalf("failed to get state rev sequence: %v", err)
		}

		for _, state := range storedStates {
			storeTestState(t, db, seq, state)
		}

		var actRes []flowstate.State
		if err := db.View(func(txn *badger.Txn) error {
			it := newLabelsIterator(txn, labels, sinceRev, reverse)
			defer it.Close()

			for ; it.Valid(); it.Next() {
				actRes = append(actRes, it.Current())

			}
			return nil
		}); err != nil {
			t.Fatalf("failed to iterate states: %v", err)
		}

		if len(actRes) != len(expRes) {
			t.Fatalf("expected %d results, got %d", len(expRes), len(actRes))
		}
		for i, state := range actRes {
			if !reflect.DeepEqual(state, expRes[i]) {
				t.Errorf("expected state %d to be %v, got %v", i, expRes[i], state)
			}
		}
	}

	// no states in db
	f(nil, map[string]string{"foo": "fooVal"}, 0, false, nil)

	// no match
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: nil,
			},
			{
				ID:     "2",
				Labels: map[string]string{},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "anotherVal"},
			},
		},
		map[string]string{"foo": "fooVal"},
		0,
		false,
		nil,
	)

	// order asc
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
		map[string]string{"foo": "fooVal"},
		0,
		false,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Rev:    2,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
	)

	// order desc (reverse)
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
		map[string]string{"foo": "fooVal"},
		math.MaxInt64,
		true,
		[]flowstate.State{
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Rev:    2,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
	)

	// since rev 2
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
		map[string]string{"foo": "fooVal"},
		2,
		false,
		[]flowstate.State{
			{
				ID:     "2",
				Rev:    2,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
	)

	// since rev 2 (reverse)
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
		map[string]string{"foo": "fooVal"},
		2,
		true,
		[]flowstate.State{
			{
				ID:     "2",
				Rev:    2,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
	)

	// select all by one label
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
		map[string]string{"foo": "fooVal"},
		0,
		false,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Rev:    2,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
	)

	// select some by one label
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "anotherVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
		map[string]string{"foo": "fooVal"},
		0,
		false,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal"},
			},
		},
	)

	// select all by several labels
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		},
		map[string]string{"foo": "fooVal", "bar": "barVal"},
		0,
		false,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Rev:    2,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		},
	)

	// select some by several labels
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "anotherVal", "bar": "barVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		},
		map[string]string{"foo": "fooVal", "bar": "barVal"},
		0,
		false,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		},
	)

	// select some by several labels
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
		},
		map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
		0,
		false,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    4,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    7,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    10,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
		},
	)

	// select some by several labels
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"foo": "anotherVal", "bar": "barVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		},
		map[string]string{"foo": "fooVal", "bar": "barVal"},
		0,
		false,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		},
	)

	// select some by several labels (reverse)
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
		},
		map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
		0,
		true,
		[]flowstate.State{
			{
				ID:     "1",
				Rev:    10,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    7,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    4,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
			},
		},
	)

	// only partially match, no results
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "ololo": "ololoVal"},
			},
		},
		map[string]string{"foo": "fooVal", "bar": "barVal", "ololo": "ololoVal"},
		0,
		false,
		[]flowstate.State{},
	)
}
