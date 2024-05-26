package flowstate

func Ended(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[stateAnnotation] == `ended`
}

func End(stateCtx *StateCtx) *EndCommand {
	return &EndCommand{
		StateCtx: stateCtx,
	}
}

type EndCommand struct {
	StateCtx *StateCtx
}

func (cmd *EndCommand) Prepare() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        ``,
		Annotations: nil,
	}
	nextTs.SetAnnotation(stateAnnotation, `ended`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
