package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestFork_WithCommit(t *testing.T) {
	d := memdriver.New()

	testcases.Fork_WithCommit(t, d, d)
}
