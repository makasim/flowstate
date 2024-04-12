package flowstate

type TaskID string

type Task struct {
	ID  TaskID `json:"id"`
	Rev int64  `json:"rev"`

	ProcessID  ProcessID `json:"process_id"`
	ProcessRev int64     `json:"process_rev"`

	DataID  DataID
	DataRev int64

	Transition Transition `json:"transition"`
}

type TaskCtx struct {
	Current   Task `json:"current"`
	Committed Task `json:"committed"`
	// Transitions between committed and current states
	Transitions []Transition `json:"transitions"`

	Process Process `json:"-"`
	Node    Node    `json:"-"`
	Data    Data    `json:"-"`
	Engine  *Engine `json:"-"`
}
