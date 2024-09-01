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

func TestQuery_GetData(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgx.Conn {
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

		d := flowstate.Data{ID: ``}
		err := q.GetData(context.Background(), conn, &d)
		require.EqualError(t, err, `id is empty`)
	})

	main.Run("RevEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := flowstate.Data{ID: `anID`, Rev: 0}
		err := q.GetData(context.Background(), conn, &d)
		require.EqualError(t, err, `rev is empty`)
	})

	main.Run("NotFound", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d1 := &flowstate.Data{ID: `anID`, Rev: 123}
		err := q.GetData(context.Background(), conn, d1)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, []byte(nil), d1.B)

		require.Equal(t, []testpgdriver.DataRow(nil), testpgdriver.FindAllData(t, conn))
	})

	main.Run("NotFoundRevMismatch", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{ID: `anID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{ID: d0.ID, Rev: 123}
		err = q.GetData(context.Background(), conn, d1)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, []byte(nil), d1.B)

		require.Equal(t, []testpgdriver.DataRow{
			{
				ID:          "anID",
				Rev:         1,
				Annotations: nil,
				B:           []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{ID: `anID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{ID: d0.ID, Rev: d0.Rev}
		err = q.GetData(context.Background(), conn, d1)
		require.NoError(t, err)
		require.Equal(t, d0, d1)

		require.Equal(t, []testpgdriver.DataRow{
			{
				ID:          "anID",
				Rev:         1,
				Annotations: nil,
				B:           []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("TxOK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{ID: `anID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		d1 := &flowstate.Data{ID: d0.ID, Rev: d0.Rev}
		err = q.GetData(context.Background(), tx, d1)
		require.NoError(t, err)
		require.Equal(t, d0, d1)

		require.NoError(t, tx.Commit(context.Background()))

		require.Equal(t, []testpgdriver.DataRow{
			{
				ID:          "anID",
				Rev:         1,
				Annotations: nil,
				B:           []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("Several", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := flowstate.Data{ID: `aFooID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, &d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := flowstate.Data{ID: `aBarID`, B: []byte(`123`)}
		err = q.InsertData(context.Background(), conn, &d1)
		require.NoError(t, err)
		require.Greater(t, d1.Rev, int64(0))

		require.Equal(t, []testpgdriver.DataRow{
			{
				ID:          "aBarID",
				Rev:         2,
				Annotations: nil,
				B:           []byte(`123`),
			},
			{
				ID:          "aFooID",
				Rev:         1,
				Annotations: nil,
				B:           []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("SeveralSameID", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := flowstate.Data{ID: `anID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, &d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		d.B = []byte(`123`)
		err = q.InsertData(context.Background(), conn, &d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		require.Equal(t, []testpgdriver.DataRow{
			{
				ID:          "anID",
				Rev:         2,
				Annotations: nil,
				B:           []byte(`123`),
			},
			{
				ID:          "anID",
				Rev:         1,
				Annotations: nil,
				B:           []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})
}
