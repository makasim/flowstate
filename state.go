package flowstate

type StateID string

type State struct {
	ID          StateID           `json:"id"`
	Rev         int64             `json:"rev"`
	Annotations map[string]string `json:"annotations"`
	Labels      map[string]string `json:"labels"`

	Transition Transition `json:"transition2"`
}

func (t *State) CopyTo(to *State) {
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

func (t *State) SetAnnotation(name, value string) {
	if t.Annotations == nil {
		t.Annotations = make(map[string]string)
	}
	t.Annotations[name] = value
}

func (t *State) SetLabel(name, value string) {
	if t.Labels == nil {
		t.Labels = make(map[string]string)
	}
	t.Labels[name] = value
}

type StateCtx struct {
	Current   State `json:"current"`
	Committed State `json:"committed"`

	// Transitions between committed and current states
	Transitions []Transition `json:"transitions2"`
}

func (t *StateCtx) CopyTo(to *StateCtx) *StateCtx {
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

	return to
}
