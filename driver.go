package flowstate

type Driver interface {
	// Init must be called by NewEngine only.
	Init(e Engine) error

	GetStateByID(cmd *GetStateByIDCommand) error
	GetStateByLabels(cmd *GetStateByLabelsCommand) error
	GetStates(cmd *GetStatesCommand) (*GetStatesResult, error)
	GetDelayedStates(cmd *GetDelayedStatesCommand) (*GetDelayedStatesResult, error)
	Delay(cmd *DelayCommand) error
	Commit(cmd *CommitCommand) error
	GetData(cmd *GetDataCommand) error
	StoreData(cmd *StoreDataCommand) error

	Flow(id TransitionID) (Flow, error)
	SetFlow(id TransitionID, flow Flow) error
}
