package flowstate

func Transit(stateCtx *StateCtx, fID FlowID) *TransitCommand {
	return &TransitCommand{
		StateCtx: stateCtx,
		FlowID:   fID,
	}
}

type TransitCommand struct {
	StateCtx *StateCtx
	FlowID   FlowID
}

func (cmd *TransitCommand) Prepare() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.FlowID,
		Annotations: nil,
	}

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
