package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestRecoveryFirstAttemptFail(t *testing.T) {
	d := memdriver.New()

	testcases.RecoveryFirstAttemptFail(t, d, d)
}
