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

var DefaultPauseDoer DoerFunc = func(cmd0 Command) error {
	cmd, ok := cmd0.(*PauseCommand)
	if !ok {
		return ErrCommandNotSupported
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.FlowID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(StateAnnotation, `paused`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
