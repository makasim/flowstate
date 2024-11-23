package flowstate

import (
	"context"
	"errors"
)

var ErrCommandNotSupported = errors.New("command not supported")

type Doer interface {
	Init(e Engine) error
	Do(cmd Command) error
	Shutdown(ctx context.Context) error
}

type DoerFunc func(cmd Command) error

func (d DoerFunc) Do(cmd Command) error {
	return d(cmd)
}

func (d DoerFunc) Init(_ Engine) error {
	return nil
}

func (d DoerFunc) Shutdown(_ context.Context) error {
	return nil
}
