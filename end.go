package flowstate

var endedAnnotation = `flowstate.ended`

func Ended(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[endedAnnotation] == `true`
}

func End(stateCtx *StateCtx) *EndCommand {
	return &EndCommand{
		StateCtx: stateCtx,
	}
}

type EndCommand struct {
	StateCtx *StateCtx
}

type Ender struct {
}

func NewEnder() *Ender {
	return &Ender{}
}

func (d *Ender) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*EndCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        ``,
		Annotations: nil,
	}
	nextTs.SetAnnotation(endedAnnotation, `true`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil, nil
}
