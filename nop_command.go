package flowstate

func Nop(taskCtx *TaskCtx) *NopCommand {
	return &NopCommand{
		TaskCtx: taskCtx,
	}
}

type NopCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *NopCommand) Prepare() error {
	return nil
}
