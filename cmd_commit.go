package flowstate

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommittableCommand interface {
	CommittableStateCtx() *StateCtx
}

type CommitCommand struct {
	command
	Commands []Command
}

func CommitStateCtx(stateCtx *StateCtx) *CommitStateCtxCommand {
	return &CommitStateCtxCommand{
		StateCtx: stateCtx,
	}
}

type CommitStateCtxCommand struct {
	command
	StateCtx *StateCtx
}

func (cmd *CommitStateCtxCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}
