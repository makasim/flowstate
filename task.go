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

	Annotations map[string]string `json:"annotations"`
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

func (t *Task) SetAnnotation(name, value string) {
	if t.Annotations == nil {
		t.Annotations = make(map[string]string)
	}
	t.Annotations[name] = value
}

type TaskCtx struct {
	Current   Task    `json:"current"`
	Committed Task    `json:"committed"`
	Process   Process `json:"process"`
	Node      Node    `json:"node"`
	Data      Data    `json:"data"`

	// Transitions between committed and current states
	Transitions []Transition `json:"transitions"`

	Engine *Engine `json:"-"`
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
