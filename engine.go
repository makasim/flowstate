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
	d  Driver
	br behaviorRegistry
}

func NewEngine(d Driver, br behaviorRegistry) *Engine {
	return &Engine{
		d:  d,
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

		if cmd1, ok := cmd.(*CommitCommand); ok {
			if len(cmd1.Commands) > 1 {
				return fmt.Errorf("commit command with more than one command not supported yet")
			}

			if err := e.d.Commit(cmd1.Commands...); err != nil {
				return err
			}

			cmd = cmd1.Commands[0]
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
