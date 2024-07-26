package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestWatchSinceRev(t *testing.T) {
	d := memdriver.New()

	testcases.WatchSinceRev(t, d, d)
}
