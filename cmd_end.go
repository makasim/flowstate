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

var DefaultEndDoer DoerFunc = func(cmd0 Command) error {
	cmd, ok := cmd0.(*EndCommand)
	if !ok {
		return ErrCommandNotSupported
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        ``,
		Annotations: nil,
	}
	nextTs.SetAnnotation(StateAnnotation, `ended`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
