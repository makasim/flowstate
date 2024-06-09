package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestThreeConsequentNodes(t *testing.T) {
	d := memdriver.New()

	usecase.ThreeConsequentNodes(t, d, d)
}
