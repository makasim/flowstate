package pgdriver

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_InsertData(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgxpool.Pool {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("NoBytes", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := &flowstate.Data{Blob: nil}
		err := q.InsertData(context.Background(), conn, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: nil,
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := &flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`abc`),
			},
		}, testpgdriver.FindAllData(t, conn))
	})

	main.Run("OKBinary", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := &flowstate.Data{Blob: []byte(`abc`)}
		d.SetBinary(true)
		err := q.InsertData(context.Background(), conn, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		rows := testpgdriver.FindAllData(t, conn)
		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`YWJj`),
				Annotations: map[string]string{
					"binary": "true",
				},
			},
		}, rows)

		actD, err := base64.StdEncoding.DecodeString(string(rows[0].Data))
		require.NoError(t, err)
		require.Equal(t, []byte(`abc`), actD)
	})

	main.Run("OKAnnotations", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := &flowstate.Data{
			Blob: []byte(`abc`),
			Annotations: map[string]string{
				"foo": "fooVal",
				"bar": "barVal",
			},
		}
		err := q.InsertData(context.Background(), conn, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		rows := testpgdriver.FindAllData(t, conn)
		require.Equal(t, []testpgdriver.DataRow{
			{
				Rev:  1,
				Data: []byte(`abc`),
				Annotations: map[string]string{
					"foo": "fooVal",
					"bar": "barVal",
				},
			},
		}, rows)
	})

	main.Run("TxOK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		q := &queries{}

		d := &flowstate.Data{Blob: []byte(`abc`)}
		err = q.InsertData(context.Background(), tx, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

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

		d0 := &flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{Blob: []byte(`123`)}
		err = q.InsertData(context.Background(), conn, d1)
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

		d := &flowstate.Data{Blob: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		d.Blob = []byte(`123`)
		err = q.InsertData(context.Background(), conn, d)
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
