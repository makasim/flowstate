package flowstate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
)

var ErrFlowNotFound = errors.New("flow not found")
var sessIDS = &atomic.Int64{}

type Engine interface {
	Execute(stateCtx *StateCtx) error
	Do(cmds ...Command) error
	Shutdown(ctx context.Context) error
}

type engine struct {
	d Driver
	l *slog.Logger

	wg     *sync.WaitGroup
	doneCh chan struct{}
}

func NewEngine(d Driver, l *slog.Logger) (Engine, error) {
	e := &engine{
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

func (e *engine) Execute(stateCtx *StateCtx) error {
	select {
	case <-e.doneCh:
		return fmt.Errorf("engine stopped")
	default:
		e.wg.Add(1)
		defer e.wg.Done()
	}

	sessID := sessIDS.Add(1)
	sessE := &execEngine{engine: e, sessID: sessID}
	stateCtx.sessID = sessID
	stateCtx.e = e
	stateCtx.doneCh = e.doneCh

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
		cmd0, err := f.Execute(stateCtx, sessE)
		if err != nil {
			return err
		}

		cmd0.setSessID(sessID)

		if cmd, ok := cmd0.(*ExecuteCommand); ok {
			cmd.sync = true
		}

		conflictErr := &ErrRevMismatch{}

		if err = e.doCmd(stateCtx.SessID(), cmd0); errors.As(err, conflictErr) {
			e.l.Info("engine: do conflict",
				"sess", cmd0.SessID(),
				"conflict", err.Error(),
				"id", stateCtx.Current.ID,
				"rev", stateCtx.Current.Rev,
			)
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

func (e *engine) Do(cmds ...Command) error {
	return e.do(0, cmds...)
}

func (e *engine) do(execSessID int64, cmds ...Command) error {
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

		if err := e.doCmd(execSessID, cmd); err != nil {
			return err
		}
	}

	return nil
}

func (e *engine) Shutdown(ctx context.Context) error {
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

func (e *engine) doCmd(execSessID int64, cmd0 Command) error {
	logDo(execSessID, cmd0, e.l)

	switch cmd := cmd0.(type) {
	case *TransitCommand:
		return cmd.do()
	case *PauseCommand:
		return cmd.do()
	case *ResumeCommand:
		return cmd.do()
	case *EndCommand:
		return cmd.do()
	case *NoopCommand:
		return nil
	case *SerializeCommand:
		return cmd.do()
	case *DeserializeCommand:
		return cmd.do()
	case *DereferenceDataCommand:
		return cmd.do()
	case *ReferenceDataCommand:
		return cmd.do()
	case *ExecuteCommand:
		if cmd.sync {
			return nil
		}

		stateCtx := cmd.StateCtx.CopyTo(&StateCtx{})
		go func() {
			if err := e.Execute(stateCtx); err != nil {
				e.l.Error("execute failed",
					"sess", stateCtx.SessID(),
					"error", err,
					"id", stateCtx.Current.ID,
					"rev", stateCtx.Current.Rev,
				)
			}
		}()

		return nil
	case *GetDataCommand:
		if err := cmd.prepare(); err != nil {
			return err
		}
		return e.d.GetData(cmd)
	case *StoreDataCommand:
		if err := cmd.prepare(); err != nil {
			return err
		}
		return e.d.StoreData(cmd)
	case *GetStateByIDCommand:
		if err := cmd.prepare(); err != nil {
			return err
		}
		return e.d.GetStateByID(cmd)
	case *GetStateByLabelsCommand:
		return e.d.GetStateByLabels(cmd)
	case *GetStatesCommand:
		cmd.prepare()

		res, err := e.d.GetStates(cmd)
		if err != nil {
			return err
		}
		cmd.result = res

		return nil
	case *DelayCommand:
		if err := cmd.prepare(); err != nil {
			return err
		}
		return e.d.Delay(cmd)
	case *GetDelayedStatesCommand:
		cmd.prepare()

		res, err := e.d.GetDelayedStates(cmd)
		if err != nil {
			return err
		}
		cmd.result = res

		return nil
	case *CommitStateCtxCommand:
		return fmt.Errorf("commit state ctx should be passed inside CommitCommand, not as a separate command")
	case *CommitCommand:
		return e.d.Commit(cmd)
	case *GetFlowCommand:
		return e.d.GetFlow(cmd)
	default:
		return fmt.Errorf("command %T not supported", cmd0)
	}
}

func (e *engine) getFlow(stateCtx *StateCtx) (Flow, error) {
	cmd := GetFlow(stateCtx)
	if err := e.d.GetFlow(cmd); err != nil {
		return nil, err
	}

	return cmd.Flow, nil
}

func (e *engine) continueExecution(cmd0 Command) (*StateCtx, error) {
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

type execEngine struct {
	*engine
	sessID int64
}

func (e *execEngine) Do(cmds ...Command) error {
	return e.do(e.sessID, cmds...)
}
