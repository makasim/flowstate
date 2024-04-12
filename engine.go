package flowstate

import (
	"errors"
	"fmt"
	"log"
	"time"
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

	if taskCtx.Current.ID == `` {
		return fmt.Errorf(`taskCtx.ID empty`)
	}

	for {
		if _, err := taskCtx.Process.Transition(taskCtx.Current.Transition.ID); err != nil {
			return err
		}

		n, err := taskCtx.Process.Node(taskCtx.Current.Transition.ToID)
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

		cmd0, err := b.Execute(taskCtx)
		if err != nil {
			return err
		}

		if err := cmd0.Prepare(); err != nil {
			return err
		}

		if cmd1, ok := cmd0.(*CommitCommand); ok {
			if len(cmd1.Commands) > 1 {
				return fmt.Errorf("commit command with more than one command not supported yet")
			}

			if err := e.d.Commit(cmd1.Commands...); err != nil {
				return err
			}

			cmd0 = cmd1.Commands[0]
		}

		switch cmd := cmd0.(type) {
		case *EndCommand:
			return nil
		case *TransitCommand:
			continue
		case *DeferCommand:
			return e.Defer(cmd)
		default:
			return fmt.Errorf("unknown command %T", cmd0)
		}
	}
}

func (e *Engine) Defer(cmd *DeferCommand) error {
	if err := cmd.Prepare(); err != nil {
		return err
	}

	// todo: replace naive implementation with real one
	go func() {
		t := time.NewTimer(cmd.Duration)
		defer t.Stop()

		<-t.C
		if err := e.Execute(cmd.DeferredTaskCtx); err != nil {
			log.Printf(`ERROR: %s`, err)
		}
	}()

	return nil
}
