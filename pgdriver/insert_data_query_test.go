package pgdriver

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_InsertData(main *testing.T) {
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

		d := &flowstate.Data{ID: ``}
		err := q.InsertData(context.Background(), conn, d)
		require.EqualError(t, err, `id is empty`)
	})

	main.Run("NoBytes", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := &flowstate.Data{ID: `anID`, B: nil}
		err := q.InsertData(context.Background(), conn, d)
		require.EqualError(t, err, `bytes is empty`)
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		d := &flowstate.Data{ID: `anID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

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

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		q := &queries{}

		d := &flowstate.Data{ID: `anID`, B: []byte(`abc`)}
		err = q.InsertData(context.Background(), tx, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

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

		d0 := &flowstate.Data{ID: `aFooID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d0)
		require.NoError(t, err)
		require.Greater(t, d0.Rev, int64(0))

		d1 := &flowstate.Data{ID: `aBarID`, B: []byte(`123`)}
		err = q.InsertData(context.Background(), conn, d1)
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

		d := &flowstate.Data{ID: `anID`, B: []byte(`abc`)}
		err := q.InsertData(context.Background(), conn, d)
		require.NoError(t, err)
		require.Greater(t, d.Rev, int64(0))

		d.B = []byte(`123`)
		err = q.InsertData(context.Background(), conn, d)
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
