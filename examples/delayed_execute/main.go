package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/examples"
)

func main() {
	slog.Default().Info("Example of delayed execute")

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

	slog.Default().Info("Delaying execution")

	// Delay the execution of the state for 1 minute
	// Delay works like a commit, so it is guaranteed that the state will be executed.
	err = e.Do(
		flowstate.Transit(stateCtx, `example`),
		flowstate.Delay(stateCtx, time.Second*10),
	)
	examples.HandleError(err)

	time.Sleep(time.Second * 15)
}
