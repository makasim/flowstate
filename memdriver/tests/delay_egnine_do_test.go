package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDelay_EngineDo(t *testing.T) {
	d := memdriver.New()

	testcases.Delay_EngineDo(t, d, d)
}
