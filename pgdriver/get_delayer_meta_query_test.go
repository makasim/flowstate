package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_GetDelayerMeta(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgx.Conn {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("NotFound", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		dm := &delayerMeta{}
		err := q.GetDelayerMeta(context.Background(), conn, 0, dm)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, delayerMeta{}, *dm)

		require.Equal(t, []testpgdriver.DataRow(nil), testpgdriver.FindAllData(t, conn))
	})

	main.Run("NotFoundShardMismatch", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{Shard: 0, Since: 123}))
		require.NoError(t, q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{Shard: 1, Since: 234}))

		dm := delayerMeta{}
		err := q.GetDelayerMeta(context.Background(), conn, 2, &dm)
		require.EqualError(t, err, `no rows in result set`)
		require.True(t, errors.Is(err, pgx.ErrNoRows))
		require.Equal(t, delayerMeta{}, dm)

		require.Len(t, testpgdriver.FindAllDelayMeta(t, conn), 2)
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{Shard: 0, Since: 123}))
		require.NoError(t, q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{Shard: 1, Since: 234}))

		dm := delayerMeta{}
		err := q.GetDelayerMeta(context.Background(), conn, 1, &dm)
		require.NoError(t, err)
		require.Equal(t, delayerMeta{Shard: 1, Since: 234}, dm)

		require.Len(t, testpgdriver.FindAllDelayMeta(t, conn), 2)
	})

	main.Run("TxOK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		require.NoError(t, q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{Shard: 0, Since: 123}))
		require.NoError(t, q.UpsertDelayerMeta(context.Background(), conn, delayerMeta{Shard: 1, Since: 234}))

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		dm := delayerMeta{}
		err = q.GetDelayerMeta(context.Background(), tx, 1, &dm)
		require.NoError(t, err)
		require.Equal(t, delayerMeta{Shard: 1, Since: 234}, dm)

		require.NoError(t, tx.Commit(context.Background()))

		require.Len(t, testpgdriver.FindAllDelayMeta(t, conn), 2)
	})
}
