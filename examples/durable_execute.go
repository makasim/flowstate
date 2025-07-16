package main

import (
	"fmt"
	"log/slog"

	"github.com/makasim/flowstate"
)

func main() {
	slog.Default().Info("Example of durable execute")

	e, d, tearDown := setUp()
	defer tearDown()

	err := d.SetFlow(`example`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		slog.Default().Info(fmt.Sprintf("executing state: %s", stateCtx.Current.ID))

		// put your business logic here

		// Tell then engine that the state is executed successfully
		return flowstate.Commit(flowstate.End(stateCtx)), nil
	}))
	handleError(err)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: `anID`,
		},
	}

	// After the state is commited it is guaranteed that the state will be executed
	// at least flowstate.DefaultMaxRecoveryAttempts
	err = e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `example`)))
	handleError(err)

	// Execute the state synchronously
	err = e.Execute(stateCtx)
	handleError(err)
}
