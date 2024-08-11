package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestActor(t *testing.T) {
	d := memdriver.New()

	testcases.Actor(t, d, d)
}
