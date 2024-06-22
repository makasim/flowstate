package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestCallFlow(t *testing.T) {
	d := memdriver.New()

	testcases.CallFlow(t, d, d)
}
