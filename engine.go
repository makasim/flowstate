package flowstate

import (
	"errors"
	"fmt"
	"log"
)

var ErrFlowNotFound = errors.New("flow not found")

type Engine struct {
	d Doer
}

func NewEngine(d Doer) (*Engine, error) {
	e := &Engine{
		d: d,
	}

	if err := d.Init(e); err != nil {
		return nil, fmt.Errorf("driver: init: %w", err)
	}

	return e, nil
}

func (e *Engine) Execute(stateCtx *StateCtx) error {
	if stateCtx.Current.ID == `` {
		return fmt.Errorf(`state id empty`)
	}

	for {
		if stateCtx.Current.Transition.ToID == `` {
			return fmt.Errorf(`transition to id empty`)
		}

		f, err := e.getFlow(stateCtx)
		if err != nil {
			return err
		}

		cmd0, err := f.Execute(stateCtx, e)
		if err != nil {
			return err
		}

		if cmd, ok := cmd0.(*ExecuteCommand); ok {
			cmd.sync = true
		}

		conflictErr := &ErrCommitConflict{}

		if err = e.do(cmd0); errors.As(err, conflictErr) {
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
		if err := e.do(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) Watch(rev int64, labels map[string]string) (Watcher, error) {
	cmd := GetWatcher(rev, labels)
	if err := e.Do(cmd); err != nil {
		return nil, err
	}

	return cmd.Watcher, nil
}

func (e *Engine) do(cmd0 Command) error {
	switch cmd := cmd0.(type) {
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
	default:
		return e.d.Do(cmd0)
	}
}

func (e *Engine) getFlow(stateCtx *StateCtx) (Flow, error) {
	cmd := GetFlow(stateCtx)
	if err := e.d.Do(cmd); err != nil {
		return nil, err
	}

	return cmd.Flow, nil
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
	case *DelayCommand:
		return nil, nil
	case *EndCommand:
		return nil, nil
	case *NoopCommand:
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown command 123 %T", cmd0)
	}
}
