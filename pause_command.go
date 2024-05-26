package flowstate

var stateAnnotation = `flowstate.state`

func Paused(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[stateAnnotation] == `paused`
}

func Pause(stateCtx *StateCtx, fID FlowID) *PauseCommand {
	return &PauseCommand{
		StateCtx: stateCtx,
		FlowID:   fID,
	}
}

type PauseCommand struct {
	StateCtx *StateCtx
	FlowID   FlowID
}

func (cmd *PauseCommand) Prepare() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.FlowID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(stateAnnotation, `paused`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
