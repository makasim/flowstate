package flowstate

import (
	"fmt"
	"time"
)

type Command interface {
	Prepare() error
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

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommitCommand struct {
	Commands []Command
}

func (cmd *CommitCommand) Prepare() error {
	if len(cmd.Commands) == 0 {
		return fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*CommitCommand); ok {
			return fmt.Errorf("commit command in commit command not allowed")
		}

		if err := c.Prepare(); err != nil {
			return err
		}
	}

	return nil
}

func Defer(taskCtx *TaskCtx, dur time.Duration) *DeferCommand {
	return &DeferCommand{
		OriginTaskCtx: taskCtx,
		Duration:      dur,
	}

}

var DeferAtAnnotation = `flowstate.defer.at`
var DeferDurationAnnotation = `flowstate.deferred.duration`

type DeferCommand struct {
	OriginTaskCtx   *TaskCtx
	DeferredTaskCtx *TaskCtx
	Duration        time.Duration
}

func (cmd *DeferCommand) Prepare() error {
	// todo: copy annotations
	// todo: xxx.Copy methods
	cmd.DeferredTaskCtx = &TaskCtx{
		Current:     cmd.OriginTaskCtx.Current,
		Committed:   cmd.OriginTaskCtx.Committed,
		Transitions: append([]Transition(nil), cmd.OriginTaskCtx.Transitions...),
		Process:     cmd.OriginTaskCtx.Process,
		Node:        cmd.OriginTaskCtx.Node,
		Data: Data{
			ID:    cmd.OriginTaskCtx.Data.ID,
			Rev:   cmd.OriginTaskCtx.Data.Rev,
			Bytes: append([]byte(nil), cmd.OriginTaskCtx.Data.Bytes...),
		},
		Engine: nil,
	}

	cmd.DeferredTaskCtx.Current.Transition.SetAnnotation(DeferAtAnnotation, time.Now().Format(time.RFC3339Nano))
	cmd.DeferredTaskCtx.Current.Transition.SetAnnotation(DeferDurationAnnotation, cmd.Duration.String())

	return nil
}
