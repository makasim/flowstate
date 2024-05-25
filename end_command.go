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
	cmd.TaskCtx.Transitions = append(cmd.TaskCtx.Transitions, cmd.TaskCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.TaskCtx.Current.Transition.ToID,
		ToID:        ``,
		Annotations: nil,
	}
	nextTs.SetAnnotation(endedAnnotation, `true`)

	cmd.TaskCtx.Current.Transition = nextTs

	return nil
}
