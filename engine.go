package flowstate

import (
	"errors"
	"fmt"
)

var ErrBehaviorNotFound = errors.New("behavior not found")

type behaviorRegistry interface {
	Behavior(id BehaviorID) (Behavior, error)
}

type MapBehaviorRegistry struct {
	behaviors map[BehaviorID]Behavior
}

func (r *MapBehaviorRegistry) SetBehavior(id BehaviorID, b Behavior) {
	if r.behaviors == nil {
		r.behaviors = make(map[BehaviorID]Behavior)
	}

	r.behaviors[id] = b
}

func (r *MapBehaviorRegistry) Behavior(id BehaviorID) (Behavior, error) {
	if r.behaviors == nil {
		return nil, ErrBehaviorNotFound
	}

	b, ok := r.behaviors[id]
	if !ok {
		return nil, ErrBehaviorNotFound
	}

	return b, nil
}

type Engine struct {
	br behaviorRegistry
}

func NewEngine(br behaviorRegistry) *Engine {
	return &Engine{
		br: br,
	}
}

func (e *Engine) Execute(taskCtx *TaskCtx) error {
	if taskCtx.Engine != nil {
		return fmt.Errorf("taskCtx.Engine already set")
	}
	taskCtx.Engine = e

	if taskCtx.ID == `` {
		return fmt.Errorf(`taskCtx.ID empty`)
	}

	for {
		ts, err := taskCtx.Process.Transition(taskCtx.Transition.ID)
		if err != nil {
			return err
		}
		taskCtx.Transition = ts

		n, err := taskCtx.Process.Node(taskCtx.Transition.ToID)
		if err != nil {
			return err
		}
		taskCtx.Node = n

		if n.BehaviorID == `` {
			return fmt.Errorf("behavior id empty")
		}

		b, err := e.br.Behavior(n.BehaviorID)
		if err != nil {
			return err
		}

		cmd, err := b.Execute(taskCtx)
		if err != nil {
			return err
		}

		if err := cmd.Do(); err != nil {
			return err
		}

		switch cmd.(type) {
		case *EndCommand:
			return nil
		case *TransitCommand:
			continue
		default:
			return fmt.Errorf("unknown command %T", cmd)
		}
	}
}

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
