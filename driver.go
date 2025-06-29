package flowstate

import (
	"errors"
)

var ErrCommandNotSupported = errors.New("command not supported")

type Driver interface {
	GetStateByID(cmd *GetStateByIDCommand) error
	GetStateByLabels(cmd *GetStateByLabelsCommand) error
	GetStates(cmd *GetStatesCommand) (*GetStatesResult, error)
	GetDelayedStates(cmd *GetDelayedStatesCommand) (*GetDelayedStatesResult, error)
	GetData(cmd *GetDataCommand) error
	GetFlow(cmd *GetFlowCommand) error
	Delay(cmd *DelayCommand) error
	StoreData(cmd *StoreDataCommand) error
	Commit(cmd *CommitCommand, e Engine) error
}
