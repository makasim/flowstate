package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/makasim/flowstate"
)

func main() {
	slog.Default().Info("Example of delayed execute")

	e, d, tearDown := setUp()
	defer tearDown()

	err := d.SetFlow(`example`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ flowstate.Engine) (flowstate.Command, error) {
		// The condition succeeds if the main execution timed out.
		// The delayed task wont be executed if the state was completed before the timeout.
		if flowstate.Delayed(stateCtx.Current) {
			slog.Default().Info(fmt.Sprintf("executing timeout logic: %s", stateCtx.Current.ID))

			// Put your timeout handling logic here
			return flowstate.Commit(flowstate.End(stateCtx)), nil
		}

		slog.Default().Info(fmt.Sprintf("executing business logic: %s", stateCtx.Current.ID))

		// Put your business logic here
		// Simulate timeout from time to time
		time.Sleep(time.Second * time.Duration(8+rand.Intn(6)))

		// Tell the engine that the state is completed
		return flowstate.Commit(flowstate.End(stateCtx)), nil
	}))
	handleError(err)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: `anID`,
		},
	}

	// Delay the execution of the state for 1 minute
	// Delay works like a commit, so it is guaranteed that the state will be executed.
	err = e.Do(
		flowstate.Commit(
			flowstate.Transit(stateCtx, `example`),
			flowstate.Delay(stateCtx, time.Second*10),
		),
		flowstate.Execute(stateCtx),
	)
	handleError(err)

	time.Sleep(time.Second * 15)
}
