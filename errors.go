package flowstate

import "errors"

// ErrRevMismatch is an error that indicates a revision mismatch during a commit operation.
type ErrRevMismatch struct {
	cmds     []string
	stateIDs []StateID
	errs     []error
}

func (err ErrRevMismatch) As(target interface{}) bool {
	if targetErr, ok := target.(*ErrRevMismatch); ok {
		*targetErr = err
		return true
	}

	return false
}

func (err ErrRevMismatch) Error() string {
	msg := "conflict;"
	for i := range err.cmds {
		msg += " cmd: " + err.cmds[i] + " sid: " + string(err.stateIDs[i]) + ";"
		if err.errs[i] != nil {
			msg += " err: " + err.errs[i].Error() + ";"
		}
	}

	return msg
}

func (err *ErrRevMismatch) Add(cmd string, sID StateID, cmdErr error) {
	err.cmds = append(err.cmds, cmd)
	err.stateIDs = append(err.stateIDs, sID)
	err.errs = append(err.errs, cmdErr)
}

func (err *ErrRevMismatch) TaskIDs() []StateID {
	return err.stateIDs
}

func (err *ErrRevMismatch) Contains(sID StateID) bool {
	for _, s := range err.stateIDs {
		if s == sID {
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
