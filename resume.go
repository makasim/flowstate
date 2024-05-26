package flowstate

var resumedAnnotation = `flowstate.resumed`

func Resumed(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[resumedAnnotation] == `true`
}

func Resume(stateCtx *StateCtx) *ResumeCommand {
	return &ResumeCommand{
		StateCtx: stateCtx,
	}

}

type ResumeCommand struct {
	StateCtx *StateCtx
}

type Resumer struct {
}

func NewResumer() *Resumer {
	return &Resumer{}
}

func (d *Resumer) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*ResumeCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.StateCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(resumedAnnotation, `true`)

	cmd.StateCtx.Current.Transition = nextTs

	return cmd.StateCtx, nil
}
