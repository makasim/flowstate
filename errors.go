package flowstate

import (
	"errors"
)

// ErrRevMismatch is an error that indicates a revision mismatch during a commit operation.
type ErrRevMismatch struct {
	IDS []StateID
}

func (err ErrRevMismatch) As(target interface{}) bool {
	if targetErr, ok := target.(*ErrRevMismatch); ok {
		*targetErr = err
		return true
	}

	return false
}

func (err ErrRevMismatch) Error() string {
	msg := "rev mismatch: "
	for i, id := range err.IDS {
		if i > 0 {
			msg += ", "
		}

		msg += string(id)
	}
	return msg
}

func (err *ErrRevMismatch) Add(id StateID) {
	err.IDS = append(err.IDS, id)
}

func (err *ErrRevMismatch) All() []StateID {
	return err.IDS
}

func (err *ErrRevMismatch) Contains(id StateID) bool {
	for _, s := range err.IDS {
		if s == id {
			return true
		}
	}

	return false
}

func asErrRevMismatch(err error) *ErrRevMismatch {
	var revErr ErrRevMismatch
	if errors.As(err, &revErr) {
		return &revErr
	}
	return nil
}

func IsErrRevMismatch(err error) bool {
	return asErrRevMismatch(err) != nil
}

func IsErrRevMismatchContains(err error, sID StateID) bool {
	revErr := asErrRevMismatch(err)
	if revErr == nil {
		return false
	}

	return revErr.Contains(sID)
}

var ErrNotFound = errors.New("state not found")
