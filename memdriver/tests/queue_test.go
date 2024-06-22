package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestQueue(t *testing.T) {
	d := memdriver.New()

	testcases.Queue(t, d, d)
}
