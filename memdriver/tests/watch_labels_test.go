package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestWatchLabels(t *testing.T) {
	d := memdriver.New()

	testcases.WatchLabels(t, d, d)
}
