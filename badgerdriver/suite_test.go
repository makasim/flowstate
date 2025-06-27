package badgerdriver_test

import (
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/badgerdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestSuite(t *testing.T) {
	s := testcases.Get(func(t testcases.TestingT) (flowstate.Driver, testcases.FlowRegistry) {
		//l, _ := testcases.NewTestLogger(t)

		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
		if err != nil {
			t.Fatalf("failed to open badger db: %v", err)
		}

		d, err := badgerdriver.New(db)
		if err != nil {
			t.Fatalf("failed to create driver: %v", err)
		}
		
		return d, d
	})

	s.Skip(t, `Delay`)
	s.Skip(t, `Cron`)
	s.Test(t)
}
