package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/examples"
)

func main() {
	slog.Default().Info("Iterate over all states example")

	e, _, _, tearDown := examples.SetUp()
	defer tearDown()

	for i := 0; i < 10; i++ {
		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: flowstate.StateID(fmt.Sprintf("anID%d", i)),
			},
		}

		err := e.Do(flowstate.Commit(flowstate.Park(stateCtx)))
		examples.HandleError(err)
	}

	iter := e.Iter(flowstate.GetStatesByLabels(nil))

	for {
		for iter.Next() {
			// Put your business logic here
			_ = iter.State()
		}
		if err := iter.Err(); err != nil {
			// Check for iteration error when iter.Next() returns false
			examples.HandleError(err)
		}

		// At this point all states till the log head have been processed
		// You can stop here or wait for new states to arrive:
		// Wait returns when new states are available or the context is done
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Minute)
		iter.Wait(waitCtx)
		waitCancel()

	}
}
