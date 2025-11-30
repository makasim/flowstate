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

type Engine struct {
	d  Driver
	fr FlowRegistry
	l  *slog.Logger

	maxRev             atomic.Int64
	maxRevCond         *sync.Cond
	MaxRevPollInterval time.Duration

	wg     *sync.WaitGroup
	doneCh chan struct{}
}

func NewEngine(d Driver, fr FlowRegistry, l *slog.Logger) (*Engine, error) {
	e := &Engine{
		d:  d,
		fr: fr,
		l:  l,

		MaxRevPollInterval: time.Second,
		wg:                 &sync.WaitGroup{},
		doneCh:             make(chan struct{}),
		maxRevCond:         sync.NewCond(&sync.Mutex{}),
	}

	if err := d.Init(e); err != nil {
		return nil, fmt.Errorf("driver: init: %w", err)
	}

	// Undone in Shutdown
	e.wg.Add(1)

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()

		e.doSyncMaxRev()
	}()

	return e, nil
}

func (e *Engine) Execute(stateCtx *StateCtx) error {
	select {
	case <-e.doneCh:
		return fmt.Errorf("engine stopped")
	default:
		e.wg.Add(1)
		defer e.wg.Done()
	}

	stateCtx.sessID = sessIDS.Add(1)
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
		cmd0, err := f.Execute(stateCtx, e)
		if err != nil {
			return err
		}

		if cmd, ok := cmd0.(*ExecuteCommand); ok {
			cmd.sync = true
		}

		conflictErr := &ErrRevMismatch{}

		if err = e.doCmd(cmd0); errors.As(err, conflictErr) {
			e.l.Info("engine: do conflict",
				"conflict", err.Error(),
				"id", stateCtx.Current.ID,
				"rev", stateCtx.Current.Rev,
			)
			return nil
		} else if err != nil {
			return err
		}

		e.maybeUpdateMaxRev(stateCtx.Current.Rev)
		e.maybeUpdateMaxRev(cmdsMaxRev(cmd0))

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
		if err := e.doCmd(cmd); err != nil {
			return err
		}
	}

	e.maybeUpdateMaxRev(cmdsMaxRev(cmds...))

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

	return nil
}

func (e *Engine) Watch(cmd *GetStatesCommand) *Watcher {
	return newWatcher(e, cmd)
}

func (e *Engine) doCmd(cmd0 Command) error {
	logCommand("engine: do", cmdsSessID(cmd0), cmd0, e.l)

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

func (e *Engine) doSyncMaxRev() {
	t := time.NewTicker(e.MaxRevPollInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			getMany := GetStatesByLabels(nil).WithLatestOnly().WithLimit(1)
			if err := e.Do(getMany); err != nil {
				e.l.Error("engine: do sync max rev: get many states: %s", "error", err)
				continue
			}

			res := getMany.MustResult()
			if len(res.States) == 0 {
				continue
			}

			e.maybeUpdateMaxRev(res.States[0].Rev)
		case <-e.doneCh:
			return
		}
	}

}

func (e *Engine) maybeUpdateMaxRev(newMaxRev int64) {
	e.maxRevCond.L.Lock()
	defer e.maxRevCond.L.Unlock()

	currMaxRev := e.maxRev.Load()
	if newMaxRev > currMaxRev {
		e.maxRev.Store(newMaxRev)
		e.maxRevCond.Broadcast()
	}
}

func cmdsSessID(cmds ...Command) int64 {
	for _, cmd0 := range cmds {
		switch cmd := cmd0.(type) {
		case *TransitCommand:
			if cmd.StateCtx.sessID != 0 {
				return cmd.StateCtx.sessID
			}
		case *ParkCommand:
			if cmd.StateCtx.sessID != 0 {
				return cmd.StateCtx.sessID
			}
		case *DelayCommand:
			if cmd.StateCtx.sessID != 0 {
				return cmd.StateCtx.sessID
			}
		case *NoopCommand:
			continue
		case *StackCommand:
			if cmd.CarrierStateCtx.sessID != 0 {
				return cmd.CarrierStateCtx.sessID
			}
		case *UnstackCommand:
			if cmd.CarrierStateCtx.sessID != 0 {
				return cmd.CarrierStateCtx.sessID
			}
		case *GetStateByIDCommand:
			continue
		case *GetStateByLabelsCommand:
			continue
		case *GetStatesCommand:
			continue
		case *GetDelayedStatesCommand:
			continue
		case *CommitCommand:
			if sid := cmdsSessID(cmd.Commands...); sid != 0 {
				return sid
			}
		case *ExecuteCommand:
			if cmd.StateCtx.sessID != 0 {
				return cmd.StateCtx.sessID
			}
		case *StoreDataCommand:
			if cmd.StateCtx.sessID != 0 {
				return cmd.StateCtx.sessID
			}
		case *GetDataCommand:
			if cmd.StateCtx.sessID != 0 {
				return cmd.StateCtx.sessID
			}
		default:
			panic(fmt.Sprintf("BUG: unknown command type %T", cmd0))
		}
	}

	return 0
}

func cmdsMaxRev(cmds ...Command) int64 {
	var maxRev int64
	for _, cmd0 := range cmds {
		switch cmd := cmd0.(type) {
		case *TransitCommand:
			maxRev = max(maxRev, cmd.StateCtx.Current.Rev)
		case *ParkCommand:
			maxRev = max(maxRev, cmd.StateCtx.Current.Rev)
		case *DelayCommand:
			maxRev = max(maxRev, cmd.StateCtx.Current.Rev)
		case *NoopCommand:
			continue
		case *StackCommand:
			maxRev = max(maxRev, cmd.CarrierStateCtx.Current.Rev)
			maxRev = max(maxRev, cmd.StackedStateCtx.Current.Rev)
		case *UnstackCommand:
			maxRev = max(maxRev, cmd.CarrierStateCtx.Current.Rev)
			maxRev = max(maxRev, cmd.UnstackStateCtx.Current.Rev)
		case *GetStateByIDCommand:
			continue
		case *GetStateByLabelsCommand:
			continue
		case *GetStatesCommand:
			continue
		case *GetDelayedStatesCommand:
			continue
		case *CommitCommand:
			maxRev = max(maxRev, cmdsMaxRev(cmd.Commands...))
		case *ExecuteCommand:
			maxRev = max(maxRev, cmd.StateCtx.Current.Rev)
		case *StoreDataCommand:
			maxRev = max(maxRev, cmd.StateCtx.Current.Rev)
		case *GetDataCommand:
			maxRev = max(maxRev, cmd.StateCtx.Current.Rev)
		default:
			panic(fmt.Sprintf("BUG: unknown command type %T", cmd0))
		}
	}

	return 0
}
