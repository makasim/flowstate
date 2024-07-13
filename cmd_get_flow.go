package flowstate

func GetFlow(stateCtx *StateCtx) *GetFlowCommand {
	return &GetFlowCommand{
		StateCtx: stateCtx,
	}
}

type GetFlowCommand struct {
	command
	StateCtx *StateCtx

	// Result
	Flow Flow
}
