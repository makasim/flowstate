package flowstate

var endedAnnotation = `flowstate.ended`

func Ended(taskCtx *TaskCtx) bool {
	return taskCtx.Current.Transition.Annotations[endedAnnotation] == `true`
}

func End(taskCtx *TaskCtx) *EndCommand {
	return &EndCommand{
		TaskCtx: taskCtx,
	}
}

type EndCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *EndCommand) Prepare() error {
	cmd.TaskCtx.Current.Transition.SetAnnotation(endedAnnotation, `true`)
	return nil
}
