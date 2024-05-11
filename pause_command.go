package flowstate

func Pause(taskCtx *TaskCtx, tsID TransitionID) *PauseCommand {
	return &PauseCommand{
		TaskCtx:      taskCtx,
		TransitionID: tsID,
	}
}

var PausedAnnotation = `flowstate.paused`

type PauseCommand struct {
	TaskCtx      *TaskCtx
	TransitionID TransitionID
}

func (cmd *PauseCommand) Prepare() error {
	if err := Transit(cmd.TaskCtx, cmd.TransitionID).Prepare(); err != nil {
		return err
	}

	cmd.TaskCtx.Current.Transition.SetAnnotation(PausedAnnotation, `true`)
	return nil
}
