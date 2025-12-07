package flowstate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

var ErrFlowNotFound = errors.New("flow not found")
var sessIDS = &atomic.Int64{}

type Engine interface {
	Execute(stateCtx *StateCtx) error
	Do(cmds ...Command) error
	Shutdown(ctx context.Context) error
}

type engine struct {
	d  *cacheDriver
	fr FlowRegistry
	l  *slog.Logger

	wg     *sync.WaitGroup
	doneCh chan struct{}
}

func NewEngine(d Driver, fr FlowRegistry, l *slog.Logger) (Engine, error) {
	e := &engine{
		d:  newCacheDriver(d, 1000, l),
		fr: fr,
		l:  l,

		wg:     &sync.WaitGroup{},
		doneCh: make(chan struct{}),
	}

	if err := d.Init(e); err != nil {
		return nil, fmt.Errorf("driver: init: %w", err)
	}

	e.wg.Add(1)

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.d.getHead(time.Millisecond*100, time.Second*5, e.doneCh)
	}()

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

		if stateCtx.Current.Transition.To == `` {
			return fmt.Errorf(`transition to id empty`)
		}

		f, err := e.fr.Flow(stateCtx.Current.Transition.To)
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

		if err = e.doCmd(stateCtx.sessID, cmd0); errors.As(err, conflictErr) {
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

	return nil
}

func (e *engine) doCmd(execSessID int64, cmd0 Command) error {
	logCommand("engine: do", execSessID, cmd0, e.l)

	switch cmd := cmd0.(type) {
	case *TransitCommand:
		return cmd.Do()
	case *ParkCommand:
		return cmd.Do()
	case *NoopCommand:
		return nil
	case *StackCommand:
		return cmd.Do()
	case *UnstackCommand:
		return cmd.Do()
	case *StoreDataCommand:
		if store, err := cmd.Prepare(); err != nil {
			return err
		} else if !store {
			return nil
		}
		if err := e.d.StoreData(cmd); err != nil {
			return err
		}
		cmd.post()
		return nil
	case *GetDataCommand:
		if get, err := cmd.Prepare(); err != nil {
			return err
		} else if !get {
			return nil
		}
		return e.d.GetData(cmd)
	case *ExecuteCommand:
		if cmd.sync {
			return nil
		}

		stateCtx := cmd.StateCtx.CopyTo(&StateCtx{})
		for n, d := range cmd.StateCtx.Datas {
			stateCtx.SetData(n, d.CopyTo(&Data{}))
		}

		go func() {
			if err := e.Execute(stateCtx); err != nil {
				e.l.Error("execute failed",
					"sess", stateCtx.sessID,
					"error", err,
					"id", stateCtx.Current.ID,
					"rev", stateCtx.Current.Rev,
				)
			}
		}()

		return nil
	case *GetStateByIDCommand:
		if err := cmd.Prepare(); err != nil {
			return err
		}
		return e.d.GetStateByID(cmd)
	case *GetStateByLabelsCommand:
		return e.d.GetStateByLabels(cmd)
	case *GetStatesCommand:
		cmd.Prepare()

		if err := e.d.GetStates(cmd); err != nil {
			return err
		}
		if cmd.Result == nil {
			return fmt.Errorf("get states command contains nil result")
		}

		return nil
	case *DelayCommand:
		if err := cmd.Prepare(); err != nil {
			return err
		}
		return e.d.Delay(cmd)
	case *GetDelayedStatesCommand:
		cmd.Prepare()

		if err := e.d.GetDelayedStates(cmd); err != nil {
			return err
		}
		if cmd.Result == nil {
			return fmt.Errorf("get delayed states command contains nil result")
		}

		return nil
	case *CommitCommand:
		if len(cmd.Commands) == 0 {
			return fmt.Errorf("no commands to commit")
		}

		for _, subCmd := range cmd.Commands {
			if _, ok := subCmd.(*CommitCommand); ok {
				return fmt.Errorf("commit command not allowed inside another commit")
			}
			if _, ok := subCmd.(*ExecuteCommand); ok {
				return fmt.Errorf("execute command not allowed inside commit")
			}
			if _, ok := subCmd.(*GetDelayedStatesCommand); ok {
				return fmt.Errorf("get delayed states command not allowed inside commit")
			}
			if _, ok := subCmd.(*GetStatesCommand); ok {
				return fmt.Errorf("get states command not allowed inside commit")
			}
		}

		return e.d.Commit(cmd)
	default:
		return fmt.Errorf("command %T not supported", cmd0)
	}
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
	case *DelayCommand:
		return nil, nil
	case *ParkCommand:
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
