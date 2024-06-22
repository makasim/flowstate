package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestRecoveryAlwaysFail(t *testing.T) {
	d := memdriver.New()

	usecase.RecoveryAlwaysFail(t, d, d)
}
