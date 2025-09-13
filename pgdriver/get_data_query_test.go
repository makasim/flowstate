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

func TestQuery_GetData(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgxpool.Pool {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("RevEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := flowstate.Data{}
		err := q.GetData(context.Background(), conn, 0, &d)
		require.EqualError(t, err, `rev is empty`)
	})

	main.Run("NotFound", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d1 := &flowstate.Data{}
		err := q.GetData(context.Background(), conn, 123, d1)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, []byte(nil), d1.Blob)

		require.Equal(t, []testpgdriver.DataRow(nil), testpgdriver.FindAllData(t, conn))
	})

	main.Run("NotFoundRevMismatch", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{}
		err = q.GetData(context.Background(), conn, 123, d1)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, []byte(nil), d1.Blob)

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{}
		err = q.GetData(context.Background(), conn, d0.Rev, d1)
		require.NoError(t, err)
		require.Equal(t, d0, d1)

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("OEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{
			Blob: nil,
		}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{}
		err = q.GetData(context.Background(), conn, d0.Rev, d1)
		require.NoError(t, err)
		require.Equal(t, d0, d1)

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: nil,
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("OKAnnotations", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{
			Blob: []byte(`abc`),
			Annotations: map[string]string{
				"foo": "fooVal",
				"bar": "barVal",
			},
		}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{}
		err = q.GetData(context.Background(), conn, d0.Rev, d1)
		require.NoError(t, err)
		require.Equal(t, d0, d1)

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`abc`),
				Annotations: map[string]string{
					"foo": "fooVal",
					"bar": "barVal",
				},
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("OKBinary", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{Blob: []byte(`abc`)}
		d0.SetBinary(true)
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{}
		err = q.GetData(context.Background(), conn, d0.Rev, d1)
		require.NoError(t, err)
		require.Equal(t, d0, d1)

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`YWJj`),
				Annotations: map[string]string{
					"binary": "true",
				},
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("TxOK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := &flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		d1 := &flowstate.Data{}
		err = q.GetData(context.Background(), tx, d0.Rev, d1)
		require.NoError(t, err)
		require.Equal(t, d0, d1)

		require.NoError(t, tx.Commit(context.Background()))

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("Several", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d0 := flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, &d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := flowstate.Data{Blob: []byte(`123`)}
		err = q.InsertData(context.Background(), conn, &d1)
		require.NoError(t, err)
		require.Greater(t, d1.Rev, int64(0))

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  2,
				Data: []byte(`123`),
			},
			{
				Rev:  1,
				Data: []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("SeveralSameID", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, &d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		d.Blob = []byte(`123`)
		err = q.InsertData(context.Background(), conn, &d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  2,
				Data: []byte(`123`),
			},
			{
				Rev:  1,
				Data: []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})
}
