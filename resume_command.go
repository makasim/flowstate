package flowstate

func Resumed(taskCtx *TaskCtx) bool {
	return taskCtx.Current.Transition.Annotations[resumedAnnotation] == `true`
}

func Resume(taskCtx *TaskCtx) *ResumeCommand {
	return &ResumeCommand{
		TaskCtx: taskCtx,
	}

}

var resumedAnnotation = `flowstate.resumed`

type ResumeCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *ResumeCommand) Prepare() error {
	if err := Transit(cmd.TaskCtx, cmd.TaskCtx.Current.Transition.ID).Prepare(); err != nil {
		return err
	}

	cmd.TaskCtx.Current.Transition.SetAnnotation(resumedAnnotation, `true`)
	return nil
}
