package memdriver_test

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestSuite(t *testing.T) {
	s := testcases.Get(func(t testcases.TestingT) (flowstate.Doer, testcases.FlowRegistry) {
		d := memdriver.New()
		return d, d
	})

	s.Test(t)
}
