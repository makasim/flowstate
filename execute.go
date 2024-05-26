package flowstate

import (
	"errors"
	"fmt"
	"log"
)

func Execute(stateCtx *StateCtx) *ExecuteCommand {
	return &ExecuteCommand{
		StateCtx: stateCtx,
	}
}

type ExecuteCommand struct {
	StateCtx *StateCtx

	sync bool
}

func (cmd *ExecuteCommand) Prepare() error {
	return nil
}

type Executor struct {
	doer Doer
}

func NewExecutor(doer Doer) *Executor {
	return &Executor{
		doer: doer,
	}
}

func (e *Executor) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*ExecuteCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	stateCtx := cmd.StateCtx
	stateCtx.Doer = e.doer

	if stateCtx.Current.ID == `` {
		return nil, fmt.Errorf(`state id empty`)
	}

	for {
		if stateCtx.Current.Transition.ToID == `` {
			return nil, fmt.Errorf(`transition to id empty`)
		}
		f, err := e.getFlow(stateCtx)
		if err != nil {
			return nil, err
		}
		cmd0, err := f.Execute(stateCtx)
		if err != nil {
			return nil, err
		}

		conflictErr := &ErrCommitConflict{}

		stateCtx, err = e.doer.Do(cmd0)
		if errors.As(err, conflictErr) {
			log.Printf("INFO: engine: execute: %s\n", conflictErr)
			return nil, nil
		} else if err != nil {
			return nil, err
		} else if stateCtx != nil {
			continue
		}

		return nil, nil
	}
}

func (e *Executor) getFlow(stateCtx *StateCtx) (Flow, error) {
	cmd := GetFlow(stateCtx)
	if _, err := e.doer.Do(cmd); err != nil {
		return nil, err
	}

	return cmd.Flow, nil
}
