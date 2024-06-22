package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestCondition(t *testing.T) {
	d := memdriver.New()

	testcases.Condition(t, d, d)
}
