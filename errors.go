package flowstate

type ErrCommitConflict struct {
	cmds     []string
	stateIDs []StateID
	errs     []error
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
		msg += " cmd: " + err.cmds[i] + " tid: " + string(err.stateIDs[i]) + ";"
		if err.errs[i] != nil {
			msg += " err: " + err.errs[i].Error() + ";"
		}
	}

	return msg
}

func (err *ErrCommitConflict) Add(cmd string, sID StateID, cmdErr error) {
	err.cmds = append(err.cmds, cmd)
	err.stateIDs = append(err.stateIDs, sID)
	err.errs = append(err.errs, cmdErr)
}

func (err *ErrCommitConflict) TaskIDs() []StateID {
	return err.stateIDs
}

func (err *ErrCommitConflict) Contains(tid StateID) bool {
	for _, t := range err.stateIDs {
		if t == tid {
			return true
		}
	}

	return false
}
