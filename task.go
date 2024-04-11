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
	Task
	Committed Task

	Process Process
	Node    Node
	Data    Data
	Engine  *Engine

	UncommittedTransitions []Transition
}
