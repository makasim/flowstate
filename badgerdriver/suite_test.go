package badgerdriver_test

import (
	"context"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/badgerdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestSuite(t *testing.T) {
	s := testcases.Get(func(t *testing.T) flowstate.Driver {
		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLoggingLevel(2))
		if err != nil {
			t.Fatalf("failed to open badger db: %v", err)
		}
		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Fatalf("failed to close badger db: %v", err)
			}
		})

		d, err := badgerdriver.New(db)
		if err != nil {
			t.Fatalf("failed to create driver: %v", err)
		}
		t.Cleanup(func() {
			if err := d.Shutdown(context.Background()); err != nil {
				t.Fatalf("failed to close driver: %v", err)
			}
		})

		return d
	})

	s.Test(t)
}
