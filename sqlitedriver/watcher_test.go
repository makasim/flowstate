package sqlitedriver_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/stretchr/testify/require"
)

func TestWatcher(main *testing.T) {
	setUp := func(t *testing.T) (*sqlitedriver.Driver, *sql.DB) {
		db, err := sql.Open("sqlite3", `:memory:`)
		require.NoError(t, err)
		db.SetMaxOpenConns(1)

		d := sqlitedriver.New(db)

		t.Cleanup(func() {
			require.NoError(t, d.Shutdown(context.Background()))
			require.NoError(t, db.Close())
		})

		_, err = flowstate.NewEngine(d)
		require.NoError(t, err)

		return d, db
	}

	main.Run("EmptyDB", func(t *testing.T) {
		d, _ := setUp(t)

		cmd := flowstate.Watch(map[string]string{"foo": "fooVal"})
		err := d.Do(cmd)
		require.NoError(t, err)
		require.NotNil(t, cmd.Listener)

		w := cmd.Listener
		defer w.Close()

		timeoutT := time.NewTimer(time.Millisecond * 200)

		select {
		case <-w.Listen():
			t.Fatal("unexpected watch")
		case <-timeoutT.C:
			break
		}
	})

	main.Run("OneStateChanges", func(t *testing.T) {
		d, db := setUp(t)

		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    1,
			Labels: map[string]string{"foo": "fooVal"},
		})
		insertStateLog(t, db, flowstate.State{
			ID:     "2",
			Rev:    2,
			Labels: map[string]string{"foo": "barVal"},
		})
		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    3,
			Labels: map[string]string{"foo": "fooVal"},
		})

		cmd := flowstate.Watch(map[string]string{"foo": "fooVal"})
		err := d.Do(cmd)
		require.NoError(t, err)
		require.NotNil(t, cmd.Listener)

		w := cmd.Listener
		defer w.Close()

		require.Equal(t, []flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal"},
			},
			{

				ID:     "1",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal"},
			},
		}, collectStates(t, w, 2))
	})

	main.Run("SeveralStateChanges", func(t *testing.T) {
		d, db := setUp(t)

		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    1,
			Labels: map[string]string{"foo": "fooVal"},
		})
		insertStateLog(t, db, flowstate.State{
			ID:     "2",
			Rev:    2,
			Labels: map[string]string{"foo": "fooVal"},
		})
		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    3,
			Labels: map[string]string{"foo": "fooVal"},
		})

		cmd := flowstate.Watch(map[string]string{"foo": "fooVal"})
		err := d.Do(cmd)
		require.NoError(t, err)
		require.NotNil(t, cmd.Listener)

		w := cmd.Listener
		defer w.Close()

		require.Equal(t, []flowstate.State{
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
				ID:     "1",
				Rev:    3,
				Labels: map[string]string{"foo": "fooVal"},
			},
		}, collectStates(t, w, 3))
	})

	main.Run("SeveralLabels", func(t *testing.T) {
		d, db := setUp(t)

		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    1,
			Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
		})

		cmd := flowstate.Watch(map[string]string{"foo": "fooVal", "bar": "barVal"})
		err := d.Do(cmd)
		require.NoError(t, err)
		require.NotNil(t, cmd.Listener)

		w := cmd.Listener
		defer w.Close()

		require.Equal(t, []flowstate.State{
			{
				ID:     "1",
				Rev:    1,
				Labels: map[string]string{"foo": "fooVal", "bar": "barVal"},
			},
		}, collectStates(t, w, 1))
	})

	main.Run("SinceRev", func(t *testing.T) {
		d, db := setUp(t)

		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    5,
			Labels: map[string]string{"foo": "fooVal"},
		})
		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    6,
			Labels: map[string]string{"foo": "fooVal"},
		})

		cmd := flowstate.Watch(map[string]string{"foo": "fooVal"}).WithSinceRev(5)
		err := d.Do(cmd)
		require.NoError(t, err)
		require.NotNil(t, cmd.Listener)

		w := cmd.Listener
		defer w.Close()

		require.Equal(t, []flowstate.State{
			{
				ID:     "1",
				Rev:    6,
				Labels: map[string]string{"foo": "fooVal"},
			},
		}, collectStates(t, w, 1))
	})

	main.Run("SinceTimeFound", func(t *testing.T) {
		d, db := setUp(t)

		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    6,
			Labels: map[string]string{"foo": "fooVal"},
		})

		cmd := flowstate.Watch(map[string]string{"foo": "fooVal"}).WithSinceTime(time.Now().Add(-time.Hour))
		err := d.Do(cmd)
		require.NoError(t, err)
		require.NotNil(t, cmd.Listener)

		w := cmd.Listener
		defer w.Close()

		require.Equal(t, []flowstate.State{
			{
				ID:     "1",
				Rev:    6,
				Labels: map[string]string{"foo": "fooVal"},
			},
		}, collectStates(t, w, 1))
	})

	main.Run("SinceTimeNotFound", func(t *testing.T) {
		d, db := setUp(t)

		insertStateLog(t, db, flowstate.State{
			ID:     "1",
			Rev:    6,
			Labels: map[string]string{"foo": "fooVal"},
		})

		cmd := flowstate.Watch(map[string]string{"foo": "fooVal"}).WithSinceTime(time.Now().Add(time.Hour))
		err := d.Do(cmd)
		require.NoError(t, err)
		require.NotNil(t, cmd.Listener)

		w := cmd.Listener
		defer w.Close()

		require.Equal(t, []flowstate.State{}, collectStates(t, w, 1))
	})
}

func collectStates(t *testing.T, w flowstate.WatchListener, limit int) []flowstate.State {
	states := make([]flowstate.State, 0, limit)

	timeoutT := time.NewTimer(time.Second)
	defer timeoutT.Stop()

loop:
	for {
		select {
		case s := <-w.Listen():
			states = append(states, s)
			if len(states) >= limit {
				break loop
			}
		case <-timeoutT.C:
			t.Fatal("timeout")
		}
	}

	return states
}

func insertStateLog(t *testing.T, db *sql.DB, state flowstate.State) {
	b, err := json.Marshal(state)
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO flowstate_state_log (id, rev, state) VALUES (?, ?, ?)",
		state.ID, state.Rev, b,
	)
	require.NoError(t, err)
}
