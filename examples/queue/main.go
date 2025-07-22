package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/examples"
)

func main() {
	slog.Default().Info("Queue example")

	e, fr, _, tearDown := examples.SetUp()
	defer tearDown()

	err := fr.SetFlow(`example`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ flowstate.Engine) (flowstate.Command, error) {
		slog.Default().Info("Executing state: " + string(stateCtx.Current.ID))
		// Tell the engine that the state is completed
		return flowstate.Commit(flowstate.End(stateCtx)), nil
	}))
	examples.HandleError(err)

	for i := 0; i < 10; i++ {
		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: flowstate.StateID(fmt.Sprintf("anID%d", i)),
				Transition: flowstate.Transition{
					To: `example`,
				},
				Labels: map[string]string{
					// a label is used to identify the queue name
					"queue": "example",
				},
			},
		}

		// Queue the states
		err = e.Do(flowstate.Commit(flowstate.Pause(stateCtx)))
		examples.HandleError(err)
	}

	w := flowstate.NewWatcher(e, time.Second, flowstate.GetStatesByLabels(map[string]string{
		"queue": "example",
	}))

	for i := 0; i < 10; i++ {
		state := <-w.Next()
		stateCtx := state.CopyToCtx(&flowstate.StateCtx{})

		// you can use state.Rev to track progress
		// you store the progress in another state

		err = e.Do(
			flowstate.Commit(flowstate.Resume(stateCtx)),
			// Uncomment to execute states in parallel
			// flowstate.Execute(stateCtx)
		)
		examples.HandleError(err)

		// Process states sequentially, in the order they were queued
		err = e.Execute(stateCtx)
		if err != nil {
			slog.Default().Error(fmt.Sprintf("Error executing state %s: %v", state.ID, err))
		}
	}
}
