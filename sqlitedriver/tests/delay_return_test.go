package tests

import (
	"context"
	"database/sql"
	"testing"

	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/makasim/flowstate/usecase"
	"github.com/stretchr/testify/require"
)

func TestDelay_Return(t *testing.T) {
	db, err := sql.Open("sqlite3", `:memory:`)
	require.NoError(t, err)
	defer db.Close()

	d := sqlitedriver.New(db)
	defer d.Shutdown(context.Background())

	usecase.Delay_Return(t, d, d)
}
