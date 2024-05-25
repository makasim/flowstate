package flowstate

import (
	"errors"
	"fmt"
	"log"
	"time"
)

var ErrFlowNotFound = errors.New("flow not found")

type flowRegistry interface {
	Flow(id FlowID) (Flow, error)
}

type MapFlowRegistry struct {
	flows map[FlowID]Flow
}

func (r *MapFlowRegistry) SetFlow(id FlowID, b Flow) {
	if r.flows == nil {
		r.flows = make(map[FlowID]Flow)
	}

	r.flows[id] = b
}

func (r *MapFlowRegistry) Flow(id FlowID) (Flow, error) {
	if r.flows == nil {
		return nil, ErrFlowNotFound
	}

	b, ok := r.flows[id]
	if !ok {
		return nil, ErrFlowNotFound
	}

	return b, nil
}

type Engine struct {
	d  Driver
	br flowRegistry
}

func NewEngine(d Driver, br flowRegistry) *Engine {
	return &Engine{
		d:  d,
		br: br,
	}
}

func (e *Engine) Execute(stateCtx *StateCtx) error {
	if stateCtx.Current.ID == `` {
		return fmt.Errorf(`state id empty`)
	}

	for {
		if stateCtx.Current.Transition.ToID == `` {
			return fmt.Errorf(`transition to id empty`)
		}

		b, err := e.br.Flow(stateCtx.Current.Transition.ToID)
		if err != nil {
			return err
		}

		cmd0, err := b.Execute(stateCtx, e)
		if err != nil {
			return err
		}

		conflictErr := &ErrCommitConflict{}

		stateCtx, err = e.prepareAndDo(cmd0, true)
		if errors.As(err, conflictErr) {
			log.Printf("INFO: engine: execute: %s\n", conflictErr)
			return nil
		} else if err != nil {
			return err
		} else if stateCtx != nil {
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
		stateCtx, err := e.prepareAndDo(cmd, false)
		if err != nil {
			return err
		} else if stateCtx != nil {
			go func() {
				if err := e.Execute(stateCtx); err != nil {
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

func (e *Engine) prepareAndDo(cmd0 Command, sync bool) (*StateCtx, error) {
	if err := cmd0.Prepare(); err != nil {
		return nil, err
	}

	return e.do(cmd0, sync)
}

func (e *Engine) do(cmd0 Command, sync bool) (*StateCtx, error) {
	if cmd1, ok := cmd0.(*CommitCommand); ok {
		if err := e.d.Do(cmd1.Commands...); err != nil {
			return nil, fmt.Errorf("driver: commit: %w", err)
		}

		var returnStateCtx *StateCtx
		for _, cmd0 := range cmd1.Commands {
			if stateCtx, err := e.do(cmd0, sync); err != nil {
				return nil, err
			} else if sync && stateCtx != nil && returnStateCtx == nil {
				returnStateCtx = stateCtx
			} else if stateCtx != nil {
				go func() {
					if err := e.Execute(stateCtx); err != nil {
						log.Printf("ERROR: engine: go execute: %s\n", err)
					}
				}()
			}
		}

		return returnStateCtx, nil
	}

	switch cmd := cmd0.(type) {
	case *EndCommand:
		return nil, nil
	case *TransitCommand:
		if sync {
			return cmd.StateCtx, nil
		}

		return nil, nil
	case *DeferCommand:
		// todo: replace naive implementation with real one
		go func() {
			t := time.NewTimer(cmd.Duration)
			defer t.Stop()

			<-t.C

			if err := e.Execute(cmd.DeferredStateCtx); err != nil {
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
			return cmd.StateCtx, nil
		}

		return nil, nil
	case *WatchCommand:
		return nil, e.d.Do(cmd)
	case *ForkCommand:
		return nil, nil
	case *ExecuteCommand:
		go func() {
			if err := e.Execute(cmd.StateCtx); err != nil {
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
