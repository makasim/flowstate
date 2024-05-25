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

func (cmd *ResumeCommand) Prepare() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.StateCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(resumedAnnotation, `true`)

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
