package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestFork(t *testing.T) {
	d := memdriver.New()

	testcases.Fork(t, d, d)
}
