package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestCondition(t *testing.T) {
	d := memdriver.New()

	testcases.Condition(t, d, d)
}
