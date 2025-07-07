package flowstate

func Ended(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `ended`
}

func End(stateCtx *StateCtx) *EndCommand {
	return &EndCommand{
		StateCtx: stateCtx,
	}
}

type EndCommand struct {
	command
	StateCtx *StateCtx
}

func (cmd *EndCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func (cmd *EndCommand) Do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		From:        cmd.StateCtx.Current.Transition.To,
		To:          ``,
		Annotations: nil,
	}
	nextTs.SetAnnotation(StateAnnotation, `ended`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
