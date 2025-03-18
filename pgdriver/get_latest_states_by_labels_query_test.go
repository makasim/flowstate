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

func TestQuery_GetLatestStatesByLabels(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgxpool.Pool {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("StatesSliceNil", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, nil)
		require.EqualError(t, err, `states slice len must be greater than 0`)
		require.Nil(t, ss)
	})

	main.Run("StatesSliceEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, []flowstate.State{})
		require.EqualError(t, err, `states slice len must be greater than 0`)
		require.Nil(t, ss)
	})

	main.Run("NoLabelsAll", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `1`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `3`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `4`}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1},
			{ID: `2`, Rev: 2},
			{ID: `3`, Rev: 3},
			{ID: `4`, Rev: 4},
		}, ss)
	})

	main.Run("NoLabelsAllSameState", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		state := &flowstate.State{ID: `1`}
		require.NoError(t, q.InsertState(context.Background(), conn, state))
		require.NoError(t, q.UpdateState(context.Background(), conn, state))
		require.NoError(t, q.UpdateState(context.Background(), conn, state))
		require.NoError(t, q.UpdateState(context.Background(), conn, state))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 4},
		}, ss)
	})

	main.Run("NoLabelsLimited", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `1`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `3`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `4`}))

		ss := make([]flowstate.State, 3)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1},
			{ID: `2`, Rev: 2},
			{ID: `3`, Rev: 3},
		}, ss)
	})

	main.Run("NoLabelsSinceRev", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `1`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `3`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `4`}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(2), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `3`, Rev: 3},
			{ID: `4`, Rev: 4},
		}, ss)
	})

	main.Run("NoLabelsSinceLatest", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `1`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `2`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `3`}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `4`}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(-1), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `4`, Rev: 4},
		}, ss)
	})

	main.Run("OneLabelAll", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
		}, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1, Labels: map[string]string{`foo`: `fooVal`}},
			{ID: `2`, Rev: 2, Labels: map[string]string{`foo`: `fooVal`}},
		}, ss)

		ss = make([]flowstate.State, 4)
		ss, err = q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`bar`: `barVal`,
			},
		}, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `3`, Rev: 3, Labels: map[string]string{`bar`: `barVal`}},
			{ID: `4`, Rev: 4, Labels: map[string]string{`bar`: `barVal`}},
		}, ss)
	})

	main.Run("OneLabelLimited", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))

		ss := make([]flowstate.State, 3)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
		}, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1, Labels: map[string]string{`foo`: `fooVal`}},
			{ID: `2`, Rev: 2, Labels: map[string]string{`foo`: `fooVal`}},
			{ID: `3`, Rev: 3, Labels: map[string]string{`foo`: `fooVal`}},
		}, ss)
	})

	main.Run("OneLabelSinceRev", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
		}, int64(2), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `3`, Rev: 3, Labels: map[string]string{`foo`: `fooVal`}},
			{ID: `4`, Rev: 4, Labels: map[string]string{`foo`: `fooVal`}},
		}, ss)
	})

	main.Run("OneLabelsSinceLatest", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
		}, int64(-1), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `4`, Rev: 4, Labels: map[string]string{`foo`: `fooVal`}},
		}, ss)
	})

	main.Run("TwoLabelsAll", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
			{ID: `2`, Rev: 2, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
		}, ss)
	})

	main.Run("TwoLabelsLimited", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 3)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
			{ID: `2`, Rev: 2, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
			{ID: `3`, Rev: 3, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
		}, ss)
	})

	main.Run("TwoLabelsSinceRev", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}, int64(2), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `3`, Rev: 3, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
			{ID: `4`, Rev: 4, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
		}, ss)
	})

	main.Run("TwoLabelsSinceLatest", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}, int64(-1), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `4`, Rev: 4, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
		}, ss)
	})

	main.Run("OrLabelsAll", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`bar`: `barVal`,
				`foo`: `fooVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
			{
				`bar`: `barVal`,
			},
		}, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1, Labels: map[string]string{`foo`: `fooVal`}},
			{ID: `2`, Rev: 2, Labels: map[string]string{`bar`: `barVal`}},
			{ID: `3`, Rev: 3, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
			{ID: `4`, Rev: 4, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
		}, ss)
	})

	main.Run("OrLabelsLimited", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`foo`: `fooVal`,
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 3)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
			{
				`bar`: `barVal`,
			},
		}, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1, Labels: map[string]string{`foo`: `fooVal`}},
			{ID: `2`, Rev: 2, Labels: map[string]string{`bar`: `barVal`}},
			{ID: `3`, Rev: 3, Labels: map[string]string{`foo`: `fooVal`, `bar`: `barVal`}},
		}, ss)
	})

	main.Run("OrLabelsSinceRev", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
			{
				`bar`: `barVal`,
			},
		}, int64(2), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `3`, Rev: 3, Labels: map[string]string{`foo`: `fooVal`}},
			{ID: `4`, Rev: 4, Labels: map[string]string{`bar`: `barVal`}},
		}, ss)
	})

	main.Run("OrLabelsSinceLatest", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `1`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `2`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `3`,
			Labels: map[string]string{
				`foo`: `fooVal`,
			},
		}))
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{
			ID: `4`,
			Labels: map[string]string{
				`bar`: `barVal`,
			},
		}))

		ss := make([]flowstate.State, 4)
		ss, err := q.GetLatestStatesByLabels(context.Background(), conn, []map[string]string{
			{
				`foo`: `fooVal`,
			},
			{
				`bar`: `barVal`,
			},
		}, int64(-1), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `4`, Rev: 4, Labels: map[string]string{`bar`: `barVal`}},
		}, ss)
	})

	main.Run("PreserveOrder", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		// past
		require.NoError(t, q.InsertState(context.Background(), conn, &flowstate.State{ID: `1`}))

		// still active
		tx0, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx0.Rollback(context.Background())
		require.NoError(t, q.InsertState(context.Background(), tx0, &flowstate.State{ID: `2`}))

		// still active
		tx1, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx1.Rollback(context.Background())
		require.NoError(t, q.InsertState(context.Background(), tx1, &flowstate.State{ID: `3`}))

		// commited but should not be visible
		tx2, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx2.Rollback(context.Background())
		require.NoError(t, q.InsertState(context.Background(), tx2, &flowstate.State{ID: `4`}))
		require.NoError(t, tx2.Commit(context.Background()))

		ss := make([]flowstate.State, 4)
		ss, err = q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1},
		}, ss)

		// now, we should see 2
		require.NoError(t, tx0.Commit(context.Background()))

		ss = make([]flowstate.State, 4)
		ss, err = q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1},
			{ID: `2`, Rev: 2},
		}, ss)

		// now, we should everything
		require.NoError(t, tx1.Commit(context.Background()))

		ss = make([]flowstate.State, 4)
		ss, err = q.GetLatestStatesByLabels(context.Background(), conn, nil, int64(0), time.Time{}, ss)
		require.NoError(t, err)
		require.Equal(t, []flowstate.State{
			{ID: `1`, Rev: 1},
			{ID: `2`, Rev: 2},
			{ID: `3`, Rev: 3},
			{ID: `4`, Rev: 4},
		}, ss)
	})
}
