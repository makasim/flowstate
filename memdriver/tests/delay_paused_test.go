package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDelay_Paused(t *testing.T) {
	d := memdriver.New()

	testcases.Delay_Paused(t, d, d)
}
