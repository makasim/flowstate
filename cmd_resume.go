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

func (cmd *ResumeCommand) do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		From:        cmd.StateCtx.Current.Transition.To,
		To:          cmd.StateCtx.Current.Transition.To,
		Annotations: nil,
	}
	nextTs.SetAnnotation(StateAnnotation, `resumed`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
