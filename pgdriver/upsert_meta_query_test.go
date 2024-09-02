package pgdriver

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_UpsertMetaQuery(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgx.Conn {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("Insert", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		err := q.UpsertMeta(context.Background(), conn, `aKey`, delayerMeta{
			Limit: 100,
			Since: 123,
			Until: 234,
		})
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.MetaRow{
			{
				Key:   `aKey`,
				Value: `{"limit": 100, "since": 123}`,
			},
		}, testpgdriver.FindAllMeta(t, conn))
	})

	main.Run("Update", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		err := q.UpsertMeta(context.Background(), conn, `aKey`, delayerMeta{
			Limit: 1,
			Since: 1,
			Until: 1,
		})
		require.NoError(t, err)

		err = q.UpsertMeta(context.Background(), conn, `aKey`, delayerMeta{
			Limit: 100,
			Since: 123,
			Until: 234,
		})
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.MetaRow{
			{
				Key:   `aKey`,
				Value: `{"limit": 100, "since": 123}`,
			},
		}, testpgdriver.FindAllMeta(t, conn))
	})

	main.Run("Tx", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		q := &queries{}

		err = q.UpsertMeta(context.Background(), tx, `aKey`, delayerMeta{
			Since: 123,
			Limit: 100,
		})
		require.NoError(t, err)

		require.NoError(t, tx.Commit(context.Background()))

		require.Equal(t, []testpgdriver.MetaRow{
			{
				Key:   `aKey`,
				Value: `{"limit": 100, "since": 123}`,
			},
		}, testpgdriver.FindAllMeta(t, conn))
	})

	main.Run("Several", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		err := q.UpsertMeta(context.Background(), conn, `aKey0`, delayerMeta{
			Since: 10,
			Limit: 100,
		})
		require.NoError(t, err)

		err = q.UpsertMeta(context.Background(), conn, `aKey1`, delayerMeta{
			Since: 20,
			Limit: 200,
		})
		require.NoError(t, err)

		err = q.UpsertMeta(context.Background(), conn, `aKey2`, delayerMeta{
			Since: 30,
			Limit: 300,
		})
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.MetaRow{
			{
				Key:   `aKey0`,
				Value: `{"limit": 100, "since": 10}`,
			},
			{
				Key:   `aKey1`,
				Value: `{"limit": 200, "since": 20}`,
			},
			{
				Key:   `aKey2`,
				Value: `{"limit": 300, "since": 30}`,
			},
		}, testpgdriver.FindAllMeta(t, conn))
	})
}
