package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestCallFlowWithWatch(t *testing.T) {
	d := memdriver.New()

	testcases.CallFlowWithWatch(t, d, d)
}
