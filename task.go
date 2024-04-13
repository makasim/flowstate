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

func (t *Task) CopyTo(to *Task) {
	to.ID = t.ID
	to.Rev = t.Rev
	to.ProcessID = t.ProcessID
	to.ProcessRev = t.ProcessRev
	to.DataID = t.DataID
	to.DataRev = t.DataRev

	t.Transition.CopyTo(&to.Transition)
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

func (t *TaskCtx) CopyTo(to *TaskCtx) {
	t.Current.CopyTo(&to.Current)
	t.Committed.CopyTo(&to.Committed)

	to.Transitions = to.Transitions[:len(t.Transitions)]
	for idx := range t.Transitions {
		t.Transitions[idx].CopyTo(&to.Transitions[idx])
	}

	// todo: add copyTo()
	to.Process = t.Process
	to.Node = t.Node
	to.Engine = t.Engine

	to.Data.ID = t.Data.ID
	to.Data.Rev = t.Data.Rev
	to.Data.Bytes = append(to.Data.Bytes[:0], t.Data.Bytes...)
}
