package tests

import (
	"testing"

	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/usecase"
)

func TestQueue(t *testing.T) {
	d := memdriver.New()

	usecase.Queue(t, d, d)
}
