package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDataStoreGetWithCommit(t *testing.T) {
	d := memdriver.New()

	testcases.DataStoreGetWithCommit(t, d, d)
}
