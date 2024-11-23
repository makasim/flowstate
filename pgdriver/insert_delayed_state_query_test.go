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

func TestQuery_InsertDelayedState(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgxpool.Pool {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("IDEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: ``}
		err := q.InsertDelayedState(context.Background(), conn, s, time.Now())
		require.EqualError(t, err, `id is empty`)
	})

	main.Run("ExecuteAtZero", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: ``}
		err := q.InsertDelayedState(context.Background(), conn, s, time.Time{})
		require.EqualError(t, err, `id is empty`)
	})

	main.Run(`OK`, func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		execAt := time.Unix(1234567, 0)

		s := flowstate.State{ID: `anID`}
		err := q.InsertDelayedState(context.Background(), conn, s, execAt)
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.DelayedStateRow{
			{
				ExecuteAt: int64(1234567),
				State: flowstate.State{
					ID: `anID`,
				},
				Pos: int64(1),
			},
		}, testpgdriver.FindAllDelayedStates(t, conn))
	})

	main.Run(`TxOK`, func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		q := &queries{}

		execAt := time.Unix(1234567, 0)

		s := flowstate.State{ID: `anID`}
		err = q.InsertDelayedState(context.Background(), tx, s, execAt)
		require.NoError(t, err)

		require.NoError(t, tx.Commit(context.Background()))

		require.Equal(t, []testpgdriver.DelayedStateRow{
			{
				ExecuteAt: int64(1234567),
				State: flowstate.State{
					ID: `anID`,
				},
				Pos: int64(1),
			},
		}, testpgdriver.FindAllDelayedStates(t, conn))
	})

	main.Run(`Several`, func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		execAt0 := time.Unix(12345, 0)

		s0 := flowstate.State{ID: `aFooID`}
		err := q.InsertDelayedState(context.Background(), conn, s0, execAt0)
		require.NoError(t, err)

		execAt1 := time.Unix(23456, 0)

		s1 := flowstate.State{ID: `aBarID`}
		err = q.InsertDelayedState(context.Background(), conn, s1, execAt1)
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.DelayedStateRow{
			{
				ExecuteAt: int64(12345),
				State: flowstate.State{
					ID: `aFooID`,
				},
				Pos: int64(1),
			},
			{
				ExecuteAt: int64(23456),
				State: flowstate.State{
					ID: `aBarID`,
				},
				Pos: int64(2),
			},
		}, testpgdriver.FindAllDelayedStates(t, conn))
	})
}
