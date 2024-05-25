package flowstate

var resumedAnnotation = `flowstate.resumed`

func Resumed(taskCtx *TaskCtx) bool {
	return taskCtx.Current.Transition.Annotations[resumedAnnotation] == `true`
}

func Resume(taskCtx *TaskCtx) *ResumeCommand {
	return &ResumeCommand{
		TaskCtx: taskCtx,
	}

}

type ResumeCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *ResumeCommand) Prepare() error {
	cmd.TaskCtx.Transitions = append(cmd.TaskCtx.Transitions, cmd.TaskCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.TaskCtx.Current.Transition.ToID,
		ToID:        cmd.TaskCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(resumedAnnotation, `true`)

	cmd.TaskCtx.Current.Transition = nextTs

	return nil
}
