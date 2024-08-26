package flowstate

func Resumed(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `resumed`
}

func Resume(stateCtx *StateCtx) *ResumeCommand {
	return &ResumeCommand{
		StateCtx: stateCtx,
	}
}

type ResumeCommand struct {
	command
	StateCtx *StateCtx
}

func (cmd *ResumeCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

var DefaultResumeDoer DoerFunc = func(cmd0 Command) error {
	cmd, ok := cmd0.(*ResumeCommand)
	if !ok {
		return ErrCommandNotSupported
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.StateCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(StateAnnotation, `resumed`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
