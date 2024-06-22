package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestTwoConsequentNodes(t *testing.T) {
	d := memdriver.New()

	testcases.TwoConsequentNodes(t, d, d)
}
