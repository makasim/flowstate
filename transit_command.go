package flowstate

func Transit(taskCtx *TaskCtx, bID BehaviorID) *TransitCommand {
	return &TransitCommand{
		TaskCtx:    taskCtx,
		BehaviorID: bID,
	}
}

type TransitCommand struct {
	TaskCtx    *TaskCtx
	BehaviorID BehaviorID
}

func (cmd *TransitCommand) Prepare() error {
	cmd.TaskCtx.Transitions = append(cmd.TaskCtx.Transitions, cmd.TaskCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.TaskCtx.Current.Transition.ToID,
		ToID:        cmd.BehaviorID,
		Annotations: nil,
	}

	cmd.TaskCtx.Current.Transition = nextTs

	return nil
}
