package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestTwoConsequentNodes(t *testing.T) {
	d := memdriver.New()

	usecase.TwoConsequentNodes(t, d, d)
}
