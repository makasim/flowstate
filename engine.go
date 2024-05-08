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
	// todo: not sure if this is necessary
	//if taskCtx.Engine != e {
	//	return fmt.Errorf("taskCtx.Engine not same as engine")
	//}
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

		taskCtx, err = e.prepareAndDo(cmd0, true)
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
		taskCtx, err := e.prepareAndDo(cmd, false)
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

func (e *Engine) Watch(rev int64, labels map[string]string) (Watcher, error) {
	cmd := Watch(rev, labels)
	if err := e.Do(cmd); err != nil {
		return nil, err
	}

	return cmd.Watcher, nil
}

func (e *Engine) prepareAndDo(cmd0 Command, sync bool) (*TaskCtx, error) {
	if err := cmd0.Prepare(); err != nil {
		return nil, err
	}

	return e.do(cmd0, sync)
}

func (e *Engine) do(cmd0 Command, sync bool) (*TaskCtx, error) {
	if cmd1, ok := cmd0.(*CommitCommand); ok {
		if err := e.d.Do(cmd1.Commands...); err != nil {
			return nil, fmt.Errorf("driver: commit: %w", err)
		}

		var returnTaskCtx *TaskCtx
		for _, cmd0 := range cmd1.Commands {
			if taskCtx, err := e.do(cmd0, sync); err != nil {
				return nil, err
			} else if sync && taskCtx != nil && returnTaskCtx == nil {
				returnTaskCtx = taskCtx
			} else if taskCtx != nil {
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
		if sync {
			return cmd.TaskCtx, nil
		}

		return nil, nil
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
		if sync {
			return cmd.TaskCtx, nil
		}

		return nil, nil
	case *WatchCommand:
		return nil, e.d.Do(cmd)
	case *ForkCommand:
		return nil, nil
	case *ExecuteCommand:
		go func() {
			if err := e.Execute(cmd.TaskCtx); err != nil {
				log.Printf("ERROR: engine: go execute: %s\n", err)
			}
		}()
		return nil, nil
	case *NopCommand:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown command %T", cmd0)
	}
}
