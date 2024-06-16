package tests

import (
	"database/sql"
	"testing"

	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/makasim/flowstate/usecase"
	"github.com/stretchr/testify/require"
)

func TestCallProcess(t *testing.T) {
	db, err := sql.Open("sqlite3", `:memory:`)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	defer db.Close()

	d := sqlitedriver.New(db)

	usecase.CallProcess(t, d, d)
}
