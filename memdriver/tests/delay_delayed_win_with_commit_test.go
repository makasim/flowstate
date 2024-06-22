package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDelay_DelayedWin_WithCommit(t *testing.T) {
	d := memdriver.New()

	testcases.Delay_DelayedWin_WithCommit(t, d, d)
}
