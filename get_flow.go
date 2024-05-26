package flowstate

func GetFlow(stateCtx *StateCtx) *GetFlowCommand {
	return &GetFlowCommand{
		StateCtx: stateCtx,
	}
}

type GetFlowCommand struct {
	StateCtx *StateCtx

	// Result
	Flow Flow
}

func (cmd *GetFlowCommand) Prepare() error {
	return nil
}
