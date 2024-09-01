package pgdriver

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_UpsertDelayerMetaQuery(main *testing.T) {
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

		err := q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{
			Shard: 1,
			Limit: 100,
			Since: 123,
			Until: 234,
		})
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.DelayMetaRow{
			{
				Shard: 1,
				Meta:  `{"limit": 100, "shard": 1, "since": 123}`,
			},
		}, testpgdriver.FindAllDelayMeta(t, conn))
	})

	main.Run("Update", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		err := q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{
			Shard: 1,
			Limit: 1,
			Since: 1,
			Until: 1,
		})
		require.NoError(t, err)

		err = q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{
			Shard: 1,
			Limit: 100,
			Since: 123,
			Until: 234,
		})
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.DelayMetaRow{
			{
				Shard: 1,
				Meta:  `{"limit": 100, "shard": 1, "since": 123}`,
			},
		}, testpgdriver.FindAllDelayMeta(t, conn))
	})

	main.Run("Tx", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		q := &queries{}

		err = q.UpsertDelayerMeta(context.Background(), tx, delayerMeta{
			Shard: 1,
			Since: 123,
			Limit: 100,
		})
		require.NoError(t, err)

		require.NoError(t, tx.Commit(context.Background()))

		require.Equal(t, []testpgdriver.DelayMetaRow{
			{
				Shard: 1,
				Meta:  `{"limit": 100, "shard": 1, "since": 123}`,
			},
		}, testpgdriver.FindAllDelayMeta(t, conn))
	})

	main.Run("Several", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		err := q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{
			Shard: 0,
			Since: 10,
			Limit: 100,
		})
		require.NoError(t, err)

		err = q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{
			Shard: 1,
			Since: 20,
			Limit: 200,
		})
		require.NoError(t, err)

		err = q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{
			Shard: 2,
			Since: 30,
			Limit: 300,
		})
		require.NoError(t, err)

		require.Equal(t, []testpgdriver.DelayMetaRow{
			{
				Shard: 0,
				Meta:  `{"limit": 100, "shard": 0, "since": 10}`,
			},
			{
				Shard: 1,
				Meta:  `{"limit": 200, "shard": 1, "since": 20}`,
			},
			{
				Shard: 2,
				Meta:  `{"limit": 300, "shard": 2, "since": 30}`,
			},
		}, testpgdriver.FindAllDelayMeta(t, conn))
	})
}
