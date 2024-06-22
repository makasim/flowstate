package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestSingleNode(t *testing.T) {
	d := memdriver.New()

	testcases.SingleNode(t, d, d)
}
