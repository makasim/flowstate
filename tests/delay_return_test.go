package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestDelay_Return(t *testing.T) {
	d := memdriver.New()

	usecase.Delay_Return(t, d, d)
}
