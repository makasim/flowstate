package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestRecoveryAlwaysFail(t *testing.T) {
	d := memdriver.New()

	testcases.RecoveryAlwaysFail(t, d, d)
}
