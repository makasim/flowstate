package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestRecoveryFirstAttemptFail(t *testing.T) {
	d := memdriver.New()

	usecase.RecoveryFirstAttemptFail(t, d, d)
}
