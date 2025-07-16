package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
)

func main() {
	slog.Default().Info("Example of durable execute")

	d := memdriver.New(slog.Default())

	e, err := flowstate.NewEngine(d, slog.Default())
	handleError(err)

	r, err := flowstate.NewRecoverer(e, slog.Default())
	handleError(err)
	defer r.Shutdown(context.Background())

	err = d.SetFlow(`example`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {

		// durable execute
		slog.Default().Info(fmt.Sprintf("executing state: %s", stateCtx.Current.ID))

		return flowstate.Commit(flowstate.End(stateCtx)), nil
	}))
	handleError(err)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: `anID`,
		},
	}

	err = e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `example`)))
	handleError(err)

	err = e.Execute(stateCtx)
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}
