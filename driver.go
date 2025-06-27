package flowstate

import (
	"errors"
)

var ErrCommandNotSupported = errors.New("command not supported")

type Driver interface {
	GetData(cmd *GetDataCommand) error
	StoreData(cmd *StoreDataCommand) error
	GetStateByID(cmd *GetStateByIDCommand) error
	GetStateByLabels(cmd *GetStateByLabelsCommand) error
	GetStates(cmd *GetStatesCommand) (*GetStatesResult, error)
	Delay(cmd *DelayCommand) error
	GetDelayedStates(cmd *GetDelayedStatesCommand) (*GetDelayedStatesResult, error)
	Commit(cmd *CommitCommand, e Engine) error
	GetFlow(cmd *GetFlowCommand) error
}
