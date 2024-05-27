package flowstate

import (
	"context"
	"errors"
)

var ErrCommandNotSupported = errors.New("command not supported")

type Command interface {
	Prepare() error
}

type Doer interface {
	Init(e Engine) error
	Do(cmd Command) error
	Shutdown(ctx context.Context) error
}
