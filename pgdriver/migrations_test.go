package pgdriver_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/makasim/flowstate/pgdriver"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	conn := testpgdriver.OpenFreshDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

	require.NotEmpty(t, pgdriver.Migrations)
	for i, m := range pgdriver.Migrations {
		require.NotEmpty(t, m.Desc)
		_, err := conn.Exec(context.Background(), m.SQL)
		require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
	}
}
