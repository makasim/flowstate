package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestDataFlowConfig(t *testing.T) {
	d := memdriver.New()

	testcases.DataFlowConfig(t, d, d)
}
