package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestWatchSinceTime(t *testing.T) {
	d := memdriver.New()

	testcases.WatchSinceTime(t, d, d)
}
