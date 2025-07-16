package main

import (
	"context"
	"log/slog"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
)

func setUp() (flowstate.Engine, flowstate.Driver, func()) {
	d := memdriver.New(slog.Default())

	e, err := flowstate.NewEngine(d, slog.Default())
	handleError(err)

	r, err := flowstate.NewRecoverer(e, slog.Default())
	handleError(err)

	dlr, err := flowstate.NewDelayer(e, slog.Default())
	handleError(err)

	return e, d, func() {
		err := dlr.Shutdown(context.Background())
		handleError(err)

		err = r.Shutdown(context.Background())
		handleError(err)
	}
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}
