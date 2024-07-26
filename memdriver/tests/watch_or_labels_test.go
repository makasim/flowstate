package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestWatchORLabels(t *testing.T) {
	d := memdriver.New()

	testcases.WatchORLabels(t, d, d)
}
