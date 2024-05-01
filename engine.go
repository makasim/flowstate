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

		taskCtx, err = e.prepareAndDo(cmd0)
		if errors.Is(err, ErrCommitConflict) {
			log.Println("INFO: engine: execute: commit conflict")
			return nil
		} else if err != nil {
			return err
		} else if taskCtx != nil {
			continue
		}

		return nil
	}
}

func (e *Engine) Do(cmds ...Command) error {
	if len(cmds) == 0 {
		return fmt.Errorf("no commands to do")
	}

	for _, cmd := range cmds {
		taskCtx, err := e.prepareAndDo(cmd)
		if err != nil {
			return err
		} else if taskCtx != nil {
			go func() {
				if err := e.Execute(taskCtx); err != nil {
					log.Printf("ERROR: engine: go execute: %s\n", err)
				}
			}()
		}
	}

	return nil
}

func (e *Engine) prepareAndDo(cmd0 Command) (*TaskCtx, error) {
	if err := cmd0.Prepare(); err != nil {
		return nil, err
	}

	return e.do(cmd0)
}

func (e *Engine) do(cmd0 Command) (*TaskCtx, error) {
	if cmd1, ok := cmd0.(*CommitCommand); ok {
		if err := e.d.Commit(cmd1.Commands...); err != nil {
			return nil, fmt.Errorf("driver: commit: %w", err)
		}

		var returnTaskCtx *TaskCtx
		for _, cmd0 := range cmd1.Commands {
			if taskCtx, err := e.do(cmd0); err != nil {
				return nil, err
			} else if taskCtx != nil && returnTaskCtx == nil {
				returnTaskCtx = taskCtx
			} else if taskCtx != nil && returnTaskCtx != nil {
				go func() {
					if err := e.Execute(taskCtx); err != nil {
						log.Printf("ERROR: engine: go execute: %s\n", err)
					}
				}()
			}
		}

		return returnTaskCtx, nil
	}

	switch cmd := cmd0.(type) {
	case *EndCommand:
		return nil, nil
	case *TransitCommand:
		return cmd.TaskCtx, nil
	case *DeferCommand:
		// todo: replace naive implementation with real one
		go func() {
			t := time.NewTimer(cmd.Duration)
			defer t.Stop()

			<-t.C

			if err := e.Execute(cmd.DeferredTaskCtx); err != nil {
				log.Printf(`ERROR: engine: defer: engine: execute: %s`, err)
			}
		}()

		return nil, nil
	case *PauseCommand:
		return nil, nil
	case *StackCommand:
		return nil, nil
	case *UnstackCommand:
		return nil, nil
	case *ResumeCommand:
		return cmd.TaskCtx, nil
	case *NopCommand:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown command %T", cmd0)
	}
}
