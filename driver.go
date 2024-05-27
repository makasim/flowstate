package flowstate

import "context"

type Driver interface {
	Init(e *Engine) error
	Shutdown(ctx context.Context) error
	Do(cmd Command) error
}
