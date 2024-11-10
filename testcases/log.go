package testcases

import (
	"log/slog"

	"github.com/thejerf/slogassert"
)

func newTestLogger(t TestingT) (*slog.Logger, *slogassert.Handler) {
	h := slogassert.New(t, slog.LevelWarn, nil)
	l := slog.New(h)

	return l, h
}
