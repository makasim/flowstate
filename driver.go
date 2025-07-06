package flowstate

import "time"

type Driver interface {
	GetStateByID(cmd *GetStateByIDCommand) error
	GetStateByLabels(cmd *GetStateByLabelsCommand) error
	GetData(cmd *GetDataCommand) error
	Delay(cmd *DelayCommand) error
	StoreData(cmd *StoreDataCommand) error
	Commit(cmd *CommitCommand, e Engine) error

	Flow(id FlowID) (Flow, error)
	SetFlow(id FlowID, flow Flow) error
	GetDelayedStates(since, until time.Time, offset int64, limit int) ([]DelayedState, bool, error)
	GetStates(orLabels []map[string]string, sinceRev int64, sinceTime time.Time, latestOnly bool, limit int) ([]State, bool, error)
}
