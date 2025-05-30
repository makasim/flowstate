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

		dm := delayerMeta{Since: 0}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.EqualError(t, err, `since is empty`)
		require.Nil(t, res)
	})

	main.Run("UntilEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		dm := delayerMeta{Since: 123, Until: 0}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.EqualError(t, err, `until is empty`)
		require.Nil(t, res)
	})

	main.Run("LimitEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		dm := delayerMeta{Since: 123, Until: 234, Limit: 0}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.EqualError(t, err, `limit is empty`)
		require.Nil(t, res)
	})

	main.Run("EmptyDB", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		dm := delayerMeta{Since: 1, Until: 10, Limit: 5}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.NoError(t, err)
		require.Equal(t, []delayedState{}, res)
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

		dm := delayerMeta{Since: 109, Until: 10000, Limit: 3}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.NoError(t, err)
		require.Equal(t, []delayedState{
			{
				ExecuteAt: 110,
				Pos:       int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
			{
				ExecuteAt: 111,
				Pos:       int64(3),
				State:     flowstate.State{ID: `ID3`},
			},
			{
				ExecuteAt: 112,
				Pos:       int64(4),
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

		dm := delayerMeta{Since: 109, Until: 10000, Limit: 3}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.NoError(t, err)
		require.Equal(t, []delayedState{
			{
				ExecuteAt: 110,
				Pos:       int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
			{
				ExecuteAt: 110,
				Pos:       int64(3),
				State:     flowstate.State{ID: `ID3`},
			},
			{
				ExecuteAt: 110,
				Pos:       int64(4),
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

		dm := delayerMeta{Since: 10, Until: 108, Limit: 3}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.NoError(t, err)
		require.Equal(t, []delayedState{}, res)
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

		dm := delayerMeta{Since: 108, Until: 10000, Limit: 10}
		res, err := q.GetDelayedStates(context.Background(), conn, dm)
		require.NoError(t, err)
		require.Equal(t, []delayedState{
			{
				ExecuteAt: 109,
				Pos:       int64(1),
				State:     flowstate.State{ID: `ID1`},
			},
		}, res)

		// now, we should see 2
		require.NoError(t, tx0.Commit(context.Background()))

		dm = delayerMeta{Since: 108, Until: 10000, Limit: 10}
		res, err = q.GetDelayedStates(context.Background(), conn, dm)
		require.NoError(t, err)
		require.Equal(t, []delayedState{
			{
				ExecuteAt: 109,
				Pos:       int64(1),
				State:     flowstate.State{ID: `ID1`},
			},
			{
				ExecuteAt: 110,
				Pos:       int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
		}, res)

		// now, we should everything
		require.NoError(t, tx1.Commit(context.Background()))

		dm = delayerMeta{Since: 108, Until: 10000, Limit: 10}
		res, err = q.GetDelayedStates(context.Background(), conn, dm)
		require.NoError(t, err)
		require.Equal(t, []delayedState{
			{
				ExecuteAt: 109,
				Pos:       int64(1),
				State:     flowstate.State{ID: `ID1`},
			},
			{
				ExecuteAt: 110,
				Pos:       int64(2),
				State:     flowstate.State{ID: `ID2`},
			},
			{
				ExecuteAt: 111,
				Pos:       int64(3),
				State:     flowstate.State{ID: `ID3`},
			},
			{
				ExecuteAt: 112,
				Pos:       int64(4),
				State:     flowstate.State{ID: `ID4`},
			},
		}, res)
	})
}
