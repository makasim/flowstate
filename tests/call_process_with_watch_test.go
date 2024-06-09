package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestCallProcessWithWatch(t *testing.T) {
	d := memdriver.New()

	usecase.CallProcessWithWatch(t, d, d)
}
