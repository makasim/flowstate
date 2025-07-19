package examples

import (
	"context"
	"log/slog"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
)

func SetUp() (flowstate.Engine, flowstate.FlowRegistry, flowstate.Driver, func()) {
	d := memdriver.New(slog.Default())
	fr := &flowstate.DefaultFlowRegistry{}

	e, err := flowstate.NewEngine(d, fr, slog.Default())
	HandleError(err)

	r, err := flowstate.NewRecoverer(e, slog.Default())
	HandleError(err)

	dlr, err := flowstate.NewDelayer(e, slog.Default())
	HandleError(err)

	return e, fr, d, func() {
		err := dlr.Shutdown(context.Background())
		HandleError(err)

		err = r.Shutdown(context.Background())
		HandleError(err)
	}
}

func HandleError(err error) {
	if err != nil {
		panic(err)
	}
}
