package flowstate

func Pause(taskCtx *TaskCtx) *PauseCommand {
	return &PauseCommand{
		TaskCtx: taskCtx,
	}

}

var PausedAnnotation = `flowstate.paused`

type PauseCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *PauseCommand) Prepare() error {
	if err := Transit(cmd.TaskCtx, cmd.TaskCtx.Current.Transition.ID).Prepare(); err != nil {
		return err
	}

	cmd.TaskCtx.Current.Transition.SetAnnotation(PausedAnnotation, `true`)
	return nil
}
