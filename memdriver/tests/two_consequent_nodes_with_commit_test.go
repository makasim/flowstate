package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestTwoConsequentNodesWithCommit(t *testing.T) {
	d := memdriver.New()

	testcases.TwoConsequentNodesWithCommit(t, d, d)
}
