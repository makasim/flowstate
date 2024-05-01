package flowstate

func End(taskCtx *TaskCtx) *EndCommand {
	return &EndCommand{
		TaskCtx: taskCtx,
	}
}

type EndCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *EndCommand) Prepare() error {
	// todo
	return nil
}
