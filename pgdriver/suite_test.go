package pgdriver_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/pgdriver"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/makasim/flowstate/testcases"
	"github.com/stretchr/testify/require"
)

func TestSuite(t *testing.T) {
	openDB := func(t testcases.TestingT, dsn0, dbName string) *pgxpool.Pool {
		conn := testpgdriver.OpenFreshDB(t.(*testing.T), dsn0, dbName)

		for i, m := range pgdriver.Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	s := testcases.Get(func(t testcases.TestingT) flowstate.Driver {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`, ``)

		t.Cleanup(func() {
			conn.Close()
		})

		l, _ := testcases.NewTestLogger(t)
		d := pgdriver.New(conn, l)
		return d
	})

	s.Test(t)
}
