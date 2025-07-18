# Flowstate

Flowstate is a lightweight Go library for building durable, long-running workflows. 
It provides a reliable framework for managing execution state, 
recovering from crashes or panics, and orchestrating complex flows with confidence. 
Whether you're handling retries, external calls, or multi-step business logic, 
Flowstate ensures that progress is safely persisted and never lost.

The persistant layer is customizable via drivers, allowing you to choose the best storage solution for your needs. 
Currently, Flowstate supports [in-memory](memdriver), [PostgreSQL](pgdriver), [BadgerDB](badgerdriver), [netdriver](netdriver) drivers.

## Examples

* [Durable execute](examples/durable_execute.go)
* [Delayed execute](examples/delayed_execute.go)
* [Testcases](testcases)

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
