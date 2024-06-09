package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestCallProcess(t *testing.T) {
	d := memdriver.New()

	usecase.CallProcess(t, d, d)
}
