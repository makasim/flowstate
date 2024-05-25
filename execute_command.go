package flowstate

func Execute(stateCtx *StateCtx) *ExecuteCommand {
	return &ExecuteCommand{
		StateCtx: stateCtx,
	}
}

type ExecuteCommand struct {
	StateCtx *StateCtx
}

func (cmd *ExecuteCommand) Prepare() error {
	return nil
}
