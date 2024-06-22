package tests

import (
	"database/sql"
	"testing"

	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/makasim/flowstate/testcases"
	"github.com/stretchr/testify/require"
)

func TestDelay_EngineDo(t *testing.T) {
	db, err := sql.Open("sqlite3", `:memory:`)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	defer db.Close()

	d := sqlitedriver.New(db)

	testcases.Delay_EngineDo(t, d, d)
}
