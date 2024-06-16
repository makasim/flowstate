package tests

import (
	"database/sql"
	"testing"

	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/makasim/flowstate/usecase"
	"github.com/stretchr/testify/require"
)

func TestForkJoin_LastWins(t *testing.T) {
	db, err := sql.Open("sqlite3", `:memory:`)
	require.NoError(t, err)
	defer db.Close()

	d := sqlitedriver.New(db)

	usecase.ForkJoin_LastWins(t, d, d)
}
