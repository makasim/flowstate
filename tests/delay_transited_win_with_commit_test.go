package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestDelay_TransitedWin_WithCommit(t *testing.T) {
	d := memdriver.New()

	usecase.Delay_TransitedWin_WithCommit(t, d, d)
}
