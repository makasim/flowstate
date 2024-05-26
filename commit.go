package flowstate

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommitCommand struct {
	Commands []Command

	NextStateCtxs []*StateCtx
}

// A driver must implement a command doer.
