package flowstate

func Noop(stateCtx *StateCtx) *NoopCommand {
	return &NoopCommand{
		StateCtx: stateCtx,
	}
}

type NoopCommand struct {
	StateCtx *StateCtx
}

func (cmd *NoopCommand) Prepare() error {
	return nil
}
