package flowstate

type TaskID string

type Task struct {
	ID          TaskID            `json:"id"`
	Rev         int64             `json:"rev"`
	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`

	Transition Transition `json:"transition2"`
}

func (t *Task) CopyTo(to *Task) {
	to.ID = t.ID
	to.Rev = t.Rev
	t.Transition.CopyTo(&to.Transition)

	for k, v := range t.Annotations {
		to.SetAnnotation(k, v)
	}
	for k, v := range t.Labels {
		to.SetLabel(k, v)
	}
}

func (t *Task) SetAnnotation(name, value string) {
	if t.Annotations == nil {
		t.Annotations = make(map[string]string)
	}
	t.Annotations[name] = value
}

func (t *Task) SetLabel(name, value string) {
	if t.Labels == nil {
		t.Labels = make(map[string]string)
	}
	t.Labels[name] = value
}

type TaskCtx struct {
	Current   Task `json:"current"`
	Committed Task `json:"committed"`

	// Transitions between committed and current states
	Transitions []Transition `json:"transitions2"`

	Engine *Engine `json:"-"`
}

func (t *TaskCtx) CopyTo(to *TaskCtx) *TaskCtx {
	t.Current.CopyTo(&to.Current)
	t.Committed.CopyTo(&to.Committed)

	if cap(to.Transitions) >= len(t.Transitions) {
		to.Transitions = to.Transitions[:len(t.Transitions)]
	} else {
		to.Transitions = make([]Transition, len(t.Transitions))
	}
	for idx := range t.Transitions {
		t.Transitions[idx].CopyTo(&to.Transitions[idx])
	}

	to.Engine = t.Engine

	return to
}
