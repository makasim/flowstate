package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestForkJoin_LastWins(t *testing.T) {
	d := memdriver.New()

	testcases.ForkJoin_LastWins(t, d, d)
}
