package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestWatchSinceLatest(t *testing.T) {
	d := memdriver.New()

	testcases.WatchSinceLatest(t, d, d)
}
