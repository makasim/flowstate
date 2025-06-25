package pgdriver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_GetDelayedStates(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgxpool.Pool {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("SinceEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		res, err := q.GetDelayedStates(context.Background(), conn, 0, 0, 0, 0)
		require.EqualError(t, err, `since is empty`)
		require.Nil(t, res)
	})

	main.Run("UntilEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		res, err := q.GetDelayedStates(context.Background(), conn, 123, 0, 0, 0)
		require.EqualError(t, err, `until is empty`)
		require.Nil(t, res)
	})

	main.Run("LimitEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		res, err := q.GetDelayedStates(context.Background(), conn, 123, 234, 0, 0)
		require.EqualError(t, err, `limit is empty`)
		require.Nil(t, res)
	})

	main.Run("EmptyDB", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		res, err := q.GetDelayedStates(context.Background(), conn, 1, 10, 0, 5)
		require.NoError(t, err)
		require.Equal(t, []flowstate.DelayedState{}, res)
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID1`}, time.Unix(109, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID2`}, time.Unix(110, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID3`}, time.Unix(111, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID4`}, time.Unix(112, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID5`}, time.Unix(113, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID6`}, time.Unix(114, 0)))

		res, err := q.GetDelayedStates(context.Background(), conn, 109, 10000, 0, 3)
		require.NoError(t, err)
		require.Equal(t, []flowstate.DelayedState{
			{
				ExecuteAt: time.Unix(110, 0),
				Offset:    int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
			{
				ExecuteAt: time.Unix(111, 0),
				Offset:    int64(3),
				State:     flowstate.State{ID: `ID3`},
			},
			{
				ExecuteAt: time.Unix(112, 0),
				Offset:    int64(4),
				State:     flowstate.State{ID: `ID4`},
			},
		}, res)
	})

	main.Run("SameTime", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID1`}, time.Unix(109, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID2`}, time.Unix(110, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID3`}, time.Unix(110, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID4`}, time.Unix(110, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID5`}, time.Unix(110, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID6`}, time.Unix(110, 0)))

		res, err := q.GetDelayedStates(context.Background(), conn, 109, 10000, 0, 3)
		require.NoError(t, err)
		require.Equal(t, []flowstate.DelayedState{
			{
				ExecuteAt: time.Unix(110, 0),
				Offset:    int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
			{
				ExecuteAt: time.Unix(110, 0),
				Offset:    int64(3),
				State:     flowstate.State{ID: `ID3`},
			},
			{
				ExecuteAt: time.Unix(110, 0),
				Offset:    int64(4),
				State:     flowstate.State{ID: `ID4`},
			},
		}, res)
	})

	main.Run("Future", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID1`}, time.Unix(109, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID2`}, time.Unix(110, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID3`}, time.Unix(111, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID4`}, time.Unix(112, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID5`}, time.Unix(113, 0)))
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID6`}, time.Unix(114, 0)))

		res, err := q.GetDelayedStates(context.Background(), conn, 10, 108, 0, 3)
		require.NoError(t, err)
		require.Equal(t, []flowstate.DelayedState{}, res)
	})

	main.Run("PreserveOrder", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		//visible
		require.NoError(t, q.InsertDelayedState(context.Background(), conn, flowstate.State{ID: `ID1`}, time.Unix(109, 0)))

		// still active
		tx0, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx0.Rollback(context.Background())
		require.NoError(t, q.InsertDelayedState(context.Background(), tx0, flowstate.State{ID: `ID2`}, time.Unix(110, 0)))

		// still active
		tx1, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx1.Rollback(context.Background())
		require.NoError(t, q.InsertDelayedState(context.Background(), tx1, flowstate.State{ID: `ID3`}, time.Unix(111, 0)))

		// commited but should not be visible
		tx2, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx2.Rollback(context.Background())
		require.NoError(t, q.InsertDelayedState(context.Background(), tx2, flowstate.State{ID: `ID4`}, time.Unix(112, 0)))
		require.NoError(t, tx2.Commit(context.Background()))

		res, err := q.GetDelayedStates(context.Background(), conn, 108, 10000, 0, 10)
		require.NoError(t, err)
		require.Equal(t, []flowstate.DelayedState{
			{
				ExecuteAt: time.Unix(109, 0),
				Offset:    int64(1),
				State:     flowstate.State{ID: `ID1`},
			},
		}, res)

		// now, we should see 2
		require.NoError(t, tx0.Commit(context.Background()))

		res, err = q.GetDelayedStates(context.Background(), conn, 108, 10000, 0, 10)
		require.NoError(t, err)
		require.Equal(t, []flowstate.DelayedState{
			{
				ExecuteAt: time.Unix(109, 0),
				Offset:    int64(1),
				State:     flowstate.State{ID: `ID1`},
			},
			{
				ExecuteAt: time.Unix(110, 0),
				Offset:    int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
		}, res)

		// now, we should everything
		require.NoError(t, tx1.Commit(context.Background()))

		res, err = q.GetDelayedStates(context.Background(), conn, 108, 10000, 0, 10)
		require.NoError(t, err)
		require.Equal(t, []flowstate.DelayedState{
			{
				ExecuteAt: time.Unix(109, 0),
				Offset:    int64(1),
				State:     flowstate.State{ID: `ID1`},
			},
			{
				ExecuteAt: time.Unix(110, 0),
				Offset:    int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
			{
				ExecuteAt: time.Unix(111, 0),
				Offset:    int64(3),
				State:     flowstate.State{ID: `ID3`},
			},
			{
				ExecuteAt: time.Unix(112, 0),
				Offset:    int64(4),
				State:     flowstate.State{ID: `ID4`},
			},
		}, res)
	})
}
