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

type Transiter struct {
}

func NewTransiter() *Transiter {
	return &Transiter{}
}

func (d *Transiter) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*TransitCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.FlowID,
		Annotations: nil,
	}

	cmd.StateCtx.Current.Transition = nextTs

	return cmd.StateCtx, nil
}
