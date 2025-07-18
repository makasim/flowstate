# Flowstate

Flowstate is a lightweight Go library for building durable, 
long-running workflows with built-in state persistence, crash recovery, and flow orchestration.
Whether you're managing retries, coordinating external services, or executing multi-step business logic, 
Flowstate ensures your progress is persisted and never lost, even in the face of panics, crashes, or OOMs.

## When to Use

Flowstate is a great fit when you need:

- Durable, crash-resistant business workflows
- Retry logic that survives restarts
- Flow coordination without introducing heavyweight workflow engines
- A minimal, embeddable orchestration layer for Go microservices
- Distributed orchestration with persistence

## Examples

* [Durable execute](examples/durable_execute.go)
* [Delayed execute](examples/delayed_execute.go)
* Realtime two play game - Gogame. [site](https://gogame.makasim.com/), [code](https://github.com/makasim/gogame).
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

## Contributing

Issues, feedback, and PRs are welcome!

## License

[MIT](LiCENSE)
