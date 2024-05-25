package flowstate

var pausedAnnotation = `flowstate.paused`

func Paused(taskCtx *TaskCtx) bool {
	return taskCtx.Current.Transition.Annotations[pausedAnnotation] == `true`
}

func Pause(taskCtx *TaskCtx, bID BehaviorID) *PauseCommand {
	return &PauseCommand{
		TaskCtx:    taskCtx,
		BehaviorID: bID,
	}
}

type PauseCommand struct {
	TaskCtx    *TaskCtx
	BehaviorID BehaviorID
}

func (cmd *PauseCommand) Prepare() error {
	cmd.TaskCtx.Transitions = append(cmd.TaskCtx.Transitions, cmd.TaskCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.TaskCtx.Current.Transition.ToID,
		ToID:        cmd.BehaviorID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(pausedAnnotation, `true`)

	cmd.TaskCtx.Current.Transition = nextTs

	return nil
}
