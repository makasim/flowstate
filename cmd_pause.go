package flowstate

func Paused(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `paused`
}

func Pause(stateCtx *StateCtx) *PauseCommand {
	return &PauseCommand{
		StateCtx: stateCtx,
		FlowID:   stateCtx.Current.Transition.To,
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

func (cmd *PauseCommand) do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		From:        cmd.StateCtx.Current.Transition.To,
		To:          cmd.FlowID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(StateAnnotation, `paused`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
