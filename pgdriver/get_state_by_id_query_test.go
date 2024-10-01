package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_GetStateByID_RevZero(main *testing.T) {
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

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, ``, 0, &s)
		require.EqualError(t, err, `id is empty`)
	})

	main.Run("NotFoundEmptyDB", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `anID`, 0, &s)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, flowstate.State{}, s)
	})

	main.Run("NotFound", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID3`}))

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `anID`, 0, &s)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, flowstate.State{}, s)
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		expS := flowstate.State{
			ID: `theID`,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}

		require.NoError(t, q.InsertState(context.Background(), conn, &expS))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID3`}))

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `theID`, 0, &s)
		require.NoError(t, err)
		require.Equal(t, flowstate.State{
			ID:  `theID`,
			Rev: 1,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}, s)
	})

	main.Run("TxOK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		expS := flowstate.State{
			ID: `theID`,
		}

		require.NoError(t, q.InsertState(context.Background(), conn, &expS))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID3`}))

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		s := flowstate.State{}
		err = q.GetStateByID(context.Background(), tx, `theID`, 0, &s)
		require.NoError(t, err)
		require.Equal(t, flowstate.State{
			ID:  `theID`,
			Rev: 1,
		}, s)
	})

	main.Run("OKLatest", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		expS := flowstate.State{
			ID: `theID`,
		}
		require.NoError(t, q.InsertState(context.Background(), conn, &expS))
		require.NoError(t, q.UpdateState(context.Background(), conn, &expS))

		expS = flowstate.State{
			ID:  `theID`,
			Rev: expS.Rev,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}
		require.NoError(t, q.UpdateState(context.Background(), conn, &expS))

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `theID`, 0, &s)
		require.NoError(t, err)
		require.Equal(t, flowstate.State{
			ID:  `theID`,
			Rev: 3,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}, s)
	})
}

func TestQuery_GetStateByID_RevNotZero(main *testing.T) {
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

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, ``, 123, &s)
		require.EqualError(t, err, `id is empty`)
	})

	main.Run("NotFoundEmptyDB", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `anID`, 123, &s)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, flowstate.State{}, s)
	})

	main.Run("NotFound", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID3`}))

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `anID`, 123, &s)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, flowstate.State{}, s)
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		expS := flowstate.State{
			ID: `theID`,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}

		require.NoError(t, q.InsertState(context.Background(), conn, &expS))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID3`}))

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `theID`, 1, &s)
		require.NoError(t, err)
		require.Equal(t, flowstate.State{
			ID:  `theID`,
			Rev: 1,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}, s)
	})

	main.Run("TxOK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		expS := flowstate.State{
			ID: `theID`,
		}

		require.NoError(t, q.InsertState(context.Background(), conn, &expS))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `otherID3`}))

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		s := flowstate.State{}
		err = q.GetStateByID(context.Background(), tx, `theID`, 1, &s)
		require.NoError(t, err)
		require.Equal(t, flowstate.State{
			ID:  `theID`,
			Rev: 1,
		}, s)
	})

	main.Run("OK2", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		expS := flowstate.State{
			ID: `theID`,
		}
		require.NoError(t, q.InsertState(context.Background(), conn, &expS))
		require.NoError(t, q.UpdateState(context.Background(), conn, &expS))

		expS = flowstate.State{
			ID:  `theID`,
			Rev: expS.Rev,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}
		require.NoError(t, q.UpdateState(context.Background(), conn, &expS))

		s := flowstate.State{}
		err := q.GetStateByID(context.Background(), conn, `theID`, 3, &s)
		require.NoError(t, err)
		require.Equal(t, flowstate.State{
			ID:  `theID`,
			Rev: 3,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}, s)
	})
}
