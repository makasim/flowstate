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

	//for {
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

	if err := b.Execute(taskCtx); err != nil {
		return err
	}
	//}
	return nil
}
