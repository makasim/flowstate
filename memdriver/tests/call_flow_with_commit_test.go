package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestCallFlowWithCommit(t *testing.T) {
	d := memdriver.New()

	testcases.CallFlowWithCommit(t, d, d)
}
