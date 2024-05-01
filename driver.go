package flowstate

import (
	"errors"
)

var ErrCommitConflict = errors.New("commit conflict")

type Driver interface {
	Commit(cmds ...Command) error
}
