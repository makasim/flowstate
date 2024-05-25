package flowstate

type ErrCommitConflict struct {
	cmds    []string
	taskIDs []TaskID
	errs    []error
}

func (err ErrCommitConflict) As(target interface{}) bool {
	if targetErr, ok := target.(*ErrCommitConflict); ok {
		*targetErr = err
		return true
	}

	return false
}

func (err ErrCommitConflict) Error() string {
	msg := "conflict;"
	for i := range err.cmds {
		msg += " cmd: " + err.cmds[i] + " tid: " + string(err.taskIDs[i]) + ";"
		if err.errs[i] != nil {
			msg += " err: " + err.errs[i].Error() + ";"
		}
	}

	return msg
}

func (err *ErrCommitConflict) Add(cmd string, tid TaskID, cmdErr error) {
	err.cmds = append(err.cmds, cmd)
	err.taskIDs = append(err.taskIDs, tid)
	err.errs = append(err.errs, cmdErr)
}

func (err *ErrCommitConflict) TaskIDs() []TaskID {
	return err.taskIDs
}

func (err *ErrCommitConflict) Contains(tid TaskID) bool {
	for _, t := range err.taskIDs {
		if t == tid {
			return true
		}
	}

	return false
}
