# Flowstate

The flowstate is a low-level state and workflow engine designed to provide a minimalistic set of primitives and actions. 
These primitives aim to offer the flexibility needed to build any state or workflow engine, low-code solution, or distributed execution.

Flow: A flow is like a function. It takes a state, performs an action, and returns a command.

State: A state records the transitions between flows and may hold other lifecycle-related information. States can be watched.

Engine: An engine executes commands given by flows and transitions a state between the flows.

Command: A command is an instruction to the engine to perform a particular action, usually on a state.

Data: Data is a payload that can be attached to a state. It can be anything and is not used or interpreted by the engine.

Watcher: A watcher can subscribe to state changes and stream them to a client.

![flowstate](https://github.com/user-attachments/assets/6f102e1e-1365-48bd-a5e9-6b0f3284849c)

Get you started:
```go
package main

import (
	"context"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
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

## Recovery

Flowstate guarantees that if a flow is committed, it will eventually complete either successfully on retry or forcefully when the maximum retry limit is reached. 

The recovery system automatically retries states that haven't reached a terminal state (ended or paused) within the configured time frame. 
By default, recovery checks for stalled states after 2 minutes, but this can be adjusted using `recovery.SetRetryAfter()`. 
The system performs up to 3 retry attempts by default before forcefully ending the state, configurable via `recovery.SetMaxAttempts()`.

Recovery timing is approximate (equal or greater than retry after) and depends on system load and available resources. 
The retry logic applies to all states by default unless explicitly disabled with `recovery.Disable()`.

In cluster mode, one process remains active to handle all recovery operations while others stay in standby mode, ready to take over if needed.

Example:
```golang
l := slog.New(slog.NewTextHandler(os.Stderr, nil))
d := memdriver.New()

e, err := flowstate.NewEngine(d, l)
if err != nil {
    log.Fatalf("failed to create engine: %v", err)
}

r := recovery.New(e, l)
if err := r.Init(); err != nil {
    log.Fatalf("failed to init recoverer: %v", err)
}

// do stuff with the engine

if err := r.Shutdown(context.Background()); err != nil {
    log.Fatalf("failed to shutdown recoverer: %v", err)
}
if err := e.Shutdown(context.Background()); err != nil {
    log.Fatalf("failed to shutdown engine: %v", err)
}
```

## Examples

Take a look at [testcases](testcases) for examples.
