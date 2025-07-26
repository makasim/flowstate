# Flowstate

Flowstate is a lightweight Go library for building durable, 
long-running workflows with built-in state persistence, crash recovery, and flow orchestration.
Whether you're managing retries, coordinating external services, or executing multi-step business logic, 
Flowstate ensures your progress is persisted and never lost, even in the face of panics, crashes, or OOMs.

You can even use Flowstate to power real-time applications like games. [Watch it in action](https://x.com/maksim_ka2/status/1825587227163795478).

## When to Use

Flowstate is a great fit when you need:

- Durable, crash-resistant business workflows
- Retry logic that survives restarts
- Flow coordination without introducing heavyweight workflow engines
- A minimal, embeddable orchestration layer for Go microservices
- Distributed orchestration with persistence

## Examples

* [Durable execute](examples/durable_execute/main.go)
* [Delayed execute](examples/delayed_execute/main.go)
* [Execute with timeout](examples/execute_with_timeout/main.go)
* [Queue](examples/queue/main.go)
* [Money transfer, aka state machine](examples/state_machine/main.go)
* Real-time Two-Player Game – GoGame: [Live Demo](https://gogame.makasim.com/) | [Video](https://x.com/maksim_ka2/status/1825587227163795478) | [Source Code](https://github.com/makasim/gogame).
* [Testcases](testcases)

## Install

```bash
go get github.com/makasim/flowstate
```

## Drivers

Flowstate supports multiple persistence backends via a simple driver interface:

- In-memory
- PostgreSQL
- BadgerDB
- Network driver

You can plug in your own driver by implementing the minimal `flowstate.Driver` interface.

## Usage Modes

### As a Library

Use Flowstate as an embedded Go library in your applications:

```go
package main

import (
	"log"
	"log/slog"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
)

func main() {
	d := memdriver.New(slog.Default())
	fr := &flowstate.DefaultFlowRegistry{}
	e, err := flowstate.NewEngine(d, fr, slog.Default())
	if err != nil {
		log.Fatalf("cannot create engine: %v", err)
	}

	// Your workflow logic here
}
```

### As a Server with Built-in UI

Run Flowstate as a standalone server with a web UI for monitoring and managing workflows:

```bash
go run app/flowstate.go

# Or using Docker
docker run -p 8080:8080 makasim/flowstate
```

The server works on `:8080` and provides:
- netdriver API
- netflow API
- UI

The server could be used as a driver for your applications:
```go
package main

import (
	"log"
	"log/slog"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/netdriver"
)

func main() {
	d := netdriver.New(`http://localhost:8080`)
	fr := &flowstate.DefaultFlowRegistry{}
	e, err := flowstate.NewEngine(d, fr, slog.Default())
	if err != nil {
		log.Fatalf("cannot create engine: %v", err)
	}

	// Your workflow logic here
}
```

## Contributing

Issues, feedback, and PRs are welcome!

## License

[MIT](LiCENSE)
