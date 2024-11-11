package flowstate

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"sync/atomic"
)

var ErrFlowNotFound = errors.New("flow not found")
var sessIDS = &atomic.Int64{}

type Engine struct {
	d Doer
	l *slog.Logger

	wg     *sync.WaitGroup
	doneCh chan struct{}
}

func NewEngine(d Doer, l *slog.Logger) (*Engine, error) {
	e := &Engine{
		d: d,
		l: l,

		wg:     &sync.WaitGroup{},
		doneCh: make(chan struct{}),
	}

	e.wg.Add(1)

	if err := d.Init(e); err != nil {
		return nil, fmt.Errorf("driver: init: %w", err)
	}

	return e, nil
}

func (e *Engine) Execute(stateCtx *StateCtx) error {
	select {
	case <-e.doneCh:
		return nil
	default:
		e.wg.Add(1)
		defer e.wg.Done()
	}

	sessID := sessIDS.Add(1)
	stateCtx.sessID = sessID
	stateCtx.e = e

	if stateCtx.Current.ID == `` {
		return fmt.Errorf(`state id empty`)
	}

	for {
		select {
		case <-e.doneCh:
			return nil
		default:
		}

		if stateCtx.Current.Transition.ToID == `` {
			return fmt.Errorf(`transition to id empty`)
		}

		f, err := e.getFlow(stateCtx)
		if err != nil {
			return err
		}

		logExecute(stateCtx, e.l)
		cmd0, err := f.Execute(stateCtx, e)
		if err != nil {
			return err
		}

		cmd0.setSessID(sessID)

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

	var sessID int64
	for _, cmd := range cmds {
		if cmd.SessID() == 0 {
			if sessID == 0 {
				sessID = sessIDS.Add(1)
			}

			cmd.setSessID(sessID)

			if cmtCmd, ok := cmd.(*CommitCommand); ok {
				for _, subCmd := range cmtCmd.Commands {
					subCmd.setSessID(sessID)
				}
			}
		}

		if err := e.do(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) Shutdown(ctx context.Context) error {
	select {
	case <-e.doneCh:
		return nil
	default:
		close(e.doneCh)
	}

	waitCh := make(chan struct{})

	go func() {
		e.wg.Wait()
		close(waitCh)
	}()

	// undo the Add in NewEngine
	e.wg.Done()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitCh:
	}

	return e.d.Shutdown(ctx)
}

func (e *Engine) do(cmd0 Command) error {
	logDo(cmd0, e.l)

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
	case *CommitCommand:
		return e.d.Do(cmd0)
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
