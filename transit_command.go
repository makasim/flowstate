package flowstate

import (
	"fmt"
)

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

func (cmd *TransitCommand) Prepare() error {
	nextTS, err := cmd.TaskCtx.Process.Transition(cmd.TransitionID)
	if err != nil {
		return err
	}

	if nextTS.ToID == `` {
		return fmt.Errorf("transition to id empty")
	}

	nextN, err := cmd.TaskCtx.Process.Node(nextTS.ToID)
	if err != nil {
		return err
	}

	cmd.TaskCtx.Transitions = append(cmd.TaskCtx.Transitions, cmd.TaskCtx.Current.Transition)
	cmd.TaskCtx.Current.Transition = nextTS
	cmd.TaskCtx.Node = nextN

	return nil
}
