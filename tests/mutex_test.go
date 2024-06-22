package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestMutex(t *testing.T) {
	d := memdriver.New()

	testcases.Mutex(t, d, d)
}
