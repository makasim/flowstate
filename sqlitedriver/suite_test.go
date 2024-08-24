package sqlitedriver_test

import (
	"database/sql"
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/makasim/flowstate/testcases"
	"github.com/stretchr/testify/require"
)

func TestSuite(t *testing.T) {
	s := testcases.Get(func(t testcases.TestingT) (flowstate.Doer, testcases.FlowRegistry) {
		db, err := sql.Open("sqlite3", `:memory:`)
		require.NoError(t, err)
		db.SetMaxOpenConns(1)

		t.Cleanup(func() {
			db.Close()
		})

		d := sqlitedriver.New(db)
		return d, d
	})

	s.Skip(t, "Actor")
	s.Test(t)
}
