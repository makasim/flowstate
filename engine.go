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

		if cmd, ok := cmd0.(*ExecuteCommand); ok {
			cmd.sync = true
		}

		conflictErr := &ErrCommitConflict{}

		if err = e.prepareAndDo(cmd0); errors.As(err, conflictErr) {
			log.Printf("INFO: engine: execute: %s\n", conflictErr)
			return nil
		} else if err != nil {
			return err
		}

		if nextStateCtx, err := e.continueExecution(cmd0); err != nil {
			return err
		} else if nextStateCtx != nil {
			stateCtx = nextStateCtx
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
		if err := e.prepareAndDo(cmd); err != nil {
			return err
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

func (e *Engine) prepareAndDo(cmd0 Command) error {
	if err := cmd0.Prepare(); err != nil {
		return err
	}

	return e.do(cmd0)
}

func (e *Engine) do(cmd0 Command) error {
	switch cmd := cmd0.(type) {
	case *CommitCommand:
		if err := e.d.Do(cmd.Commands...); err != nil {
			return fmt.Errorf("driver: commit: %w", err)
		}

		for _, subCmd := range cmd.Commands {
			if err := e.do(subCmd); err != nil {
				return err
			}
		}

		return nil
	case *EndCommand:
		return nil
	case *TransitCommand:
		return nil
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

		return nil
	case *PauseCommand:
		return nil
	case *StackCommand:
		return nil
	case *UnstackCommand:
		return nil
	case *ResumeCommand:
		return nil
	case *WatchCommand:
		return e.d.Do(cmd)
	case *ForkCommand:
		return nil
	case *ExecuteCommand:
		if cmd.sync {
			return nil
		}

		go func() {
			if err := e.Execute(cmd.StateCtx); err != nil {
				log.Printf("ERROR: engine: go execute: %s\n", err)
			}
		}()
		return nil
	case *NopCommand:
		return nil
	default:
		return fmt.Errorf("unknown command %T", cmd0)
	}
}

func (e *Engine) continueExecution(cmd0 Command) (*StateCtx, error) {
	switch cmd := cmd0.(type) {
	case *CommitCommand:
		if len(cmd.Commands) != 1 {
			return nil, fmt.Errorf("commit command must have exactly one command")
		}

		return e.continueExecution(cmd.Commands[0])
	case *ExecuteCommand:
		return cmd.StateCtx, nil
	case *TransitCommand:
		return cmd.StateCtx, nil
	case *ResumeCommand:
		return cmd.StateCtx, nil
	case *PauseCommand:
		return nil, nil
	case *DeferCommand:
		return nil, nil
	case *EndCommand:
		return nil, nil
	case *NopCommand:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown command 123 %T", cmd0)
	}
}
