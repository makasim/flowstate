package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_GetStateByLabels(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgx.Conn {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("LabelsNil", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, nil, 0, &s)
		require.EqualError(t, err, `labels are empty`)
	})

	main.Run("LabelsEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, map[string]string{}, 0, &s)
		require.EqualError(t, err, `labels are empty`)
	})

	main.Run("NotFoundEmptyDB", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		ls := map[string]string{
			`foo`: `fooVal`,
		}
		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, ls, 0, &s)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, flowstate.State{}, s)
	})

	main.Run("NotFound", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `otherID`,
			Labels: map[string]string{
				`foo`: `fooOtherVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `otherID2`,
			Labels: map[string]string{
				`foo`: `fooOtherVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `otherID3`,
			Labels: map[string]string{
				`foo`: `fooOtherVal`,
			},
		}))

		ls := map[string]string{
			`foo`: `fooVal`,
		}
		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, ls, 0, &s)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, flowstate.State{}, s)
	})

	main.Run("OneLabelMatch", func(t *testing.T) {
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

		ls := map[string]string{
			`fooLabelKey`: `fooLabelVal`,
		}
		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, ls, 0, &s)
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

	main.Run("SeveralLabelMatch", func(t *testing.T) {
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

		ls := map[string]string{
			`fooLabelKey`: `fooLabelVal`,
			`barLabelKey`: `barLabelVal`,
		}
		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, ls, 0, &s)
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

	main.Run("OneLabelNotMatch", func(t *testing.T) {
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

		ls := map[string]string{
			`fooLabelKey`: `fooLabelVal`,
			`barLabelKey`: `barLabelAnotherVal`,
		}
		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, ls, 0, &s)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, flowstate.State{}, s)
	})

	main.Run("OneLatest", func(t *testing.T) {
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
		require.NoError(t, q.UpdateState(context.Background(), conn, &expS))
		require.NoError(t, q.UpdateState(context.Background(), conn, &expS))

		ls := map[string]string{
			`fooLabelKey`: `fooLabelVal`,
		}
		s := flowstate.State{}
		err := q.GetStateByLabels(context.Background(), conn, ls, 0, &s)
		require.NoError(t, err)
		require.Equal(t, int64(3), s.Rev)
	})
}
