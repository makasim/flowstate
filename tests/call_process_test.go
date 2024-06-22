package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestCallProcess(t *testing.T) {
	d := memdriver.New()

	testcases.CallProcess(t, d, d)
}
