package flowstate

func Paused(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `paused`
}

func Pause(stateCtx *StateCtx) *PauseCommand {
	return &PauseCommand{
		StateCtx: stateCtx,
		FlowID:   stateCtx.Current.Transition.ToID,
	}
}

type PauseCommand struct {
	command

	StateCtx *StateCtx
	FlowID   FlowID
}

func (cmd *PauseCommand) WithTransit(fID FlowID) *PauseCommand {
	cmd.FlowID = fID
	return cmd
}

func (cmd *PauseCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}
