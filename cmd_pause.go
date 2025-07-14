package flowstate

func Paused(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `paused`
}

func Pause(stateCtx *StateCtx) *PauseCommand {
	return &PauseCommand{
		StateCtx: stateCtx,
		To:       stateCtx.Current.Transition.To,
	}
}

type PauseCommand struct {
	command

	StateCtx *StateCtx
	To       TransitionID
}

func (cmd *PauseCommand) WithTransit(to TransitionID) *PauseCommand {
	cmd.To = to
	return cmd
}

func (cmd *PauseCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func (cmd *PauseCommand) Do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		From:        cmd.StateCtx.Current.Transition.To,
		To:          cmd.To,
		Annotations: nil,
	}
	nextTs.SetAnnotation(StateAnnotation, `paused`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
