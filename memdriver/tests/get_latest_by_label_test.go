package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestGetLatestByLabel(t *testing.T) {
	d := memdriver.New()

	testcases.GetLatestByLabel(t, d, d)
}
