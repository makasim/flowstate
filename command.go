package flowstate

import "errors"

var ErrCommandNotSupported = errors.New("command not supported")

type Command interface {
	//NextStateCtx() *StateCtx
}

type Doer interface {
	Do(cmd Command) (*StateCtx, error)
}
