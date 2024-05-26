package flowstate

var pausedAnnotation = `flowstate.paused`

func Paused(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[pausedAnnotation] == `true`
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

type Pauser struct {
}

func NewPauser() *Pauser {
	return &Pauser{}
}

func (d *Pauser) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*PauseCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.FlowID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(pausedAnnotation, `true`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil, nil
}
