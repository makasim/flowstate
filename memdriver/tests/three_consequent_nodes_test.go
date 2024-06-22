package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestThreeConsequentNodes(t *testing.T) {
	d := memdriver.New()

	testcases.ThreeConsequentNodes(t, d, d)
}
