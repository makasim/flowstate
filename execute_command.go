package flowstate

func Execute(taskCtx *TaskCtx) *ExecuteCommand {
	return &ExecuteCommand{
		TaskCtx: taskCtx,
	}
}

type ExecuteCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *ExecuteCommand) Prepare() error {
	return nil
}
