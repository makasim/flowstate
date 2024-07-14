# Flowstate

The flowstate is a low-level state and workflow engine designed to provide a minimalistic set of primitives and actions. 
These primitives aim to offer the flexibility needed to build any state or workflow engine, low-code solution, or distributed execution.

Flow: A flow is like a function. It takes a state, performs an action, and returns a command.

State: A state records the transitions between flows and may hold other lifecycle-related information. States can be watched.

Engine: An engine executes commands given by flows and transitions a state between the flows.

Command: A command is an instruction to the engine to perform a particular action, usually on a state.

Data: Data is a payload that can be attached to a state. It can be anything and is not used or interpreted by the engine.

Watcher: A watcher can subscribe to state changes and stream them to a client.

```go
package main

import (
	"context"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func main() {
	d := memdriver.New()

	d.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ *flowstate.Engine) (flowstate.Command, error) {
		return flowstate.Transit(stateCtx, `second`), nil
	}))
	d.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ *flowstate.Engine) (flowstate.Command, error) {
		return flowstate.End(stateCtx), nil
	}))

	e, err := flowstate.NewEngine(d)
	if err != nil {
		panic(err)
	}
	defer e.Shutdown(context.Background())

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aTID",
		},
	}

	if err := e.Do(flowstate.Transit(stateCtx, `first`)); err != nil {
		panic(err)
	}
	if err := e.Execute(stateCtx); err != nil {
		panic(err)
	}
}


```

## Examples

Take a look at [testcases](testcases) for examples.
