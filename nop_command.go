package flowstate

func Nop(stateCtx *StateCtx) *NopCommand {
	return &NopCommand{
		StateCtx: stateCtx,
	}
}

type NopCommand struct {
	StateCtx *StateCtx
}

func (cmd *NopCommand) Prepare() error {
	return nil
}
