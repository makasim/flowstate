package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestRateLimit(t *testing.T) {
	d := memdriver.New()

	testcases.RateLimit(t, d, d)
}
