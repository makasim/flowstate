package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDelay_TransitedWin_WithCommit(t *testing.T) {
	d := memdriver.New()

	testcases.Delay_TransitedWin_WithCommit(t, d, d)
}
