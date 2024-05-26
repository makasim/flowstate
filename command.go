package flowstate

import "errors"

var ErrCommandNotSupported = errors.New("command not supported")

type Command interface {
	Prepare() error
}
