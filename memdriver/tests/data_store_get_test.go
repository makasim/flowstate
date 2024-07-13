package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDataStoreGet(t *testing.T) {
	d := memdriver.New()

	testcases.DataStoreGet(t, d, d)
}
