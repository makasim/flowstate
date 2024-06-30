package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDelay_PausedWithCommit(t *testing.T) {
	d := memdriver.New()

	testcases.Delay_PausedWithCommit(t, d, d)
}
