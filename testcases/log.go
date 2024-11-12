package testcases

import (
	"log/slog"
	"os"

	"github.com/thejerf/slogassert"
)

func NewTestLogger(t TestingT) (*slog.Logger, *slogassert.Handler) {
	var wrappedH slog.Handler
	if os.Getenv(`TEST_OUTPUT_LOG`) == `true` {
		wrappedH = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == `time` {
					return slog.Attr{}
				}
				return a
			},
		})
	}

	h := slogassert.New(t, slog.LevelDebug, wrappedH)
	l := slog.New(h)

	return l, h
}
