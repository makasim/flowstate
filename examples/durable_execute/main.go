package main

import (
	"fmt"
	"log/slog"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/examples"
)

func main() {
	slog.Default().Info("Example of durable execute")

	e, fr, _, tearDown := examples.SetUp()
	defer tearDown()

	err := fr.SetFlow(`example`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ flowstate.Engine) (flowstate.Command, error) {
		slog.Default().Info(fmt.Sprintf("executing state: %s", stateCtx.Current.ID))

		// Put your business logic here

		// Tell the engine that the state is completed
		return flowstate.Commit(flowstate.End(stateCtx)), nil
	}))
	examples.HandleError(err)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: `anID`,
		},
	}

	// After the state is commited it is guaranteed that the state will be executed
	// at least flowstate.DefaultMaxRecoveryAttempts attempts
	err = e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `example`)))
	examples.HandleError(err)

	// Execute the state synchronously
	// The engine will call the example flow.
	err = e.Execute(stateCtx)
	examples.HandleError(err)
}
