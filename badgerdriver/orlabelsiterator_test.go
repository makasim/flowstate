package badgerdriver

import (
	"context"
	"reflect"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func TestOrLabelsIterator(t *testing.T) {
	f := func(storedStates []flowstate.State, orLabels []map[string]string, sinceRev int64, reverse bool, expRes []flowstate.State) {
		t.Helper()

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

		for _, state := range storedStates {
			storeTestState(t, d, state)
		}

		var actRes []flowstate.State
		if err := db.View(func(txn *badger.Txn) error {
			it := newOrLabelsIterator(txn, orLabels, sinceRev, reverse)
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
	f(nil, []map[string]string{{"foo": "fooVal"}}, 0, false, nil)

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
		[]map[string]string{{"foo": "fooVal"}},
		0,
		false,
		nil,
	)

	// match all if no labels
	f(
		[]flowstate.State{
			{
				ID:     "id1",
				Labels: nil,
			},
			{
				ID:     "id2",
				Labels: map[string]string{},
			},
			{
				ID:     "id3",
				Labels: map[string]string{"foo": "anotherVal"},
			},
		},
		nil,
		0,
		false,
		[]flowstate.State{
			{
				ID:     "id1",
				Rev:    1,
				Labels: nil,
			},
			{
				ID:     "id2",
				Rev:    2,
				Labels: map[string]string{},
			},
			{
				ID:     "id3",
				Rev:    3,
				Labels: map[string]string{"foo": "anotherVal"},
			},
		},
	)

	// match all if no labels (reverse)
	f(
		[]flowstate.State{
			{
				ID:     "id1",
				Labels: nil,
			},
			{
				ID:     "id2",
				Labels: map[string]string{},
			},
			{
				ID:     "id3",
				Labels: map[string]string{"foo": "anotherVal"},
			},
		},
		nil,
		0,
		true,
		[]flowstate.State{
			{
				ID:     "id3",
				Rev:    3,
				Labels: map[string]string{"foo": "anotherVal"},
			},
			{
				ID:     "id2",
				Rev:    2,
				Labels: map[string]string{},
			},
			{
				ID:     "id1",
				Rev:    1,
				Labels: nil,
			},
		},
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
				Labels: map[string]string{"bar": "barVal"},
			},
		},
		[]map[string]string{{"foo": "fooVal"}, {"bar": "barVal"}},
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
				Labels: map[string]string{"bar": "barVal"},
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
				Labels: map[string]string{"bar": "barVal"},
			},
		},
		[]map[string]string{{"foo": "fooVal"}, {"bar": "barVal"}},
		0,
		true,
		[]flowstate.State{
			{
				ID:     "3",
				Rev:    3,
				Labels: map[string]string{"bar": "barVal"},
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
				Labels: map[string]string{"bar": "barVal"},
			},
		},
		[]map[string]string{{"foo": "fooVal"}, {"bar": "barVal"}},
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
				Labels: map[string]string{"bar": "barVal"},
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
				Labels: map[string]string{"bar": "barVal"},
			},
		},
		[]map[string]string{{"foo": "fooVal"}, {"bar": "barVal"}},
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

	// select all
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal", "foo": "fooVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal", "foo": "fooVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal"},
			},
		},
		[]map[string]string{{"foo": "fooVal", "bar": "barVal"}, {"bar": "barVal", "ololo": "ololoVal"}},
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
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    7,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Rev:    8,
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
		},
	)

	// select all (reverse)
	f(
		[]flowstate.State{
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal", "foo": "fooVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal", "foo": "fooVal"},
			},
			{
				ID:     "1",
				Labels: map[string]string{"foo": "fooVal"},
			},
			{
				ID:     "2",
				Labels: map[string]string{"bar": "barVal"},
			},
			{
				ID:     "3",
				Labels: map[string]string{"ololo": "ololoVal"},
			},
		},
		[]map[string]string{{"foo": "fooVal", "bar": "barVal"}, {"bar": "barVal", "ololo": "ololoVal"}},
		0,
		true,
		[]flowstate.State{
			{
				ID:     "2",
				Rev:    8,
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    7,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
			{
				ID:     "2",
				Rev:    2,
				Labels: map[string]string{"bar": "barVal", "ololo": "ololoVal"},
			},
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		},
	)
}
