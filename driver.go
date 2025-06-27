package flowstate

import (
	"context"
	"errors"
)

var ErrCommandNotSupported = errors.New("command not supported")

type Driver interface {
	Init(e Engine) error
	Shutdown(ctx context.Context) error

	GetData(cmd *GetDataCommand) error
	StoreData(cmd *StoreDataCommand) error
	GetStateByID(cmd *GetStateByIDCommand) error
	GetStateByLabels(cmd *GetStateByLabelsCommand) error
	GetStates(cmd *GetStatesCommand) (*GetStatesResult, error)
	Delay(cmd *DelayCommand) error
	GetDelayedStates(cmd *GetDelayedStatesCommand) (*GetDelayedStatesResult, error)
	Commit(cmd *CommitCommand) error
	GetFlow(cmd *GetFlowCommand) error
}
