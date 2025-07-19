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
* [Execute with delay](examples/execute_with_timeout/main.go)
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

## Contributing

Issues, feedback, and PRs are welcome!

## License

[MIT](LiCENSE)
