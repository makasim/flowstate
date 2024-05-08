package flowstate

import (
	"errors"
)

var ErrCommitConflict = errors.New("commit conflict")

type Driver interface {
	Do(cmds ...Command) error
}
