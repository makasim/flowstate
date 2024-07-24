package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestGetLatestByID(t *testing.T) {
	d := memdriver.New()

	testcases.GetLatestByID(t, d, d)
}
