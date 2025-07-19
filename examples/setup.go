package main

import (
	"context"
	"log/slog"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
)

func setUp() (flowstate.Engine, flowstate.FlowRegistry, flowstate.Driver, func()) {
	d := memdriver.New(slog.Default())
	fr := &flowstate.DefaultFlowRegistry{}

	e, err := flowstate.NewEngine(d, fr, slog.Default())
	handleError(err)

	r, err := flowstate.NewRecoverer(e, slog.Default())
	handleError(err)

	dlr, err := flowstate.NewDelayer(e, slog.Default())
	handleError(err)

	return e, fr, d, func() {
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
