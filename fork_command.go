package flowstate

import "fmt"

func Fork(taskCtx, forkedTaskCtx *TaskCtx) *ForkCommand {
	return &ForkCommand{
		TaskCtx:       taskCtx,
		ForkedTaskCtx: forkedTaskCtx,
	}

}

type ForkCommand struct {
	TaskCtx       *TaskCtx
	ForkedTaskCtx *TaskCtx
}

func (cmd *ForkCommand) Prepare() error {
	tID := cmd.ForkedTaskCtx.Current.ID

	cmd.TaskCtx.CopyTo(cmd.ForkedTaskCtx)

	cmd.ForkedTaskCtx.Current.ID = tID
	cmd.ForkedTaskCtx.Current.Rev = 0
	//cmd.ForkedTaskCtx.Current.CopyTo(&forkedTaskCtx.Committed)

	if cmd.ForkedTaskCtx.Current.ID == `` {
		return fmt.Errorf(`forked task ID empty`)
	}

	return nil
}
