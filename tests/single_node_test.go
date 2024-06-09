package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestSingleNode(t *testing.T) {
	d := memdriver.New()

	usecase.SingleNode(t, d, d)
}
