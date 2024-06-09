package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestTwoConsequentNodesWithCommit(t *testing.T) {
	d := memdriver.New()

	usecase.TwoConsequentNodesWithCommit(t, d, d)
}
