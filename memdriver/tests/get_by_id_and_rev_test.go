package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestGetByIDAndRev(t *testing.T) {
	d := memdriver.New()

	testcases.GetByIDAndRev(t, d, d)
}
