package flowstate

import "fmt"

type Command interface {
	Do() error
}

func Transit(taskCtx *TaskCtx, tsID TransitionID) *TransitCommand {
	return &TransitCommand{
		TaskCtx:      taskCtx,
		TransitionID: tsID,
	}
}

type TransitCommand struct {
	TaskCtx      *TaskCtx
	TransitionID TransitionID
}

func (cmd *TransitCommand) Do() error {
	ts, err := cmd.TaskCtx.Process.Transition(cmd.TransitionID)
	if err != nil {
		return err
	}

	if ts.ToID == `` {
		return fmt.Errorf("transition to id empty")
	}

	n, err := cmd.TaskCtx.Process.Node(ts.ToID)
	if err != nil {
		return err
	}

	cmd.TaskCtx.Transition = ts
	cmd.TaskCtx.Node = n

	return nil
}

func End(taskCtx *TaskCtx) *EndCommand {
	return &EndCommand{
		TaskCtx: taskCtx,
	}
}

type EndCommand struct {
	TaskCtx *TaskCtx
}

func (cmd *EndCommand) Do() error {
	// todo
	return nil
}

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommitCommand struct {
	Commands []Command
}

func (cmd *CommitCommand) Do() error {
	if len(cmd.Commands) == 0 {
		return fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*CommitCommand); ok {
			return fmt.Errorf("commit command in commit command not allowed")
		}

		if err := c.Do(); err != nil {
			return err
		}
	}

	return nil
}
