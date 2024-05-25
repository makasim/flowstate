package flowstate

type Transition struct {
	FromID      BehaviorID        `json:"from"`
	ToID        BehaviorID        `json:"to"`
	Annotations map[string]string `json:"annotations"`
}

func (ts *Transition) SetAnnotation(name, value string) {
	if ts.Annotations == nil {
		ts.Annotations = make(map[string]string)
	}
	ts.Annotations[name] = value
}

func (ts *Transition) CopyTo(to *Transition) {
	to.FromID = ts.FromID
	to.ToID = ts.ToID

	if len(ts.Annotations) > 0 {
		if to.Annotations == nil {
			to.Annotations = make(map[string]string)
		}

		for k, v := range ts.Annotations {
			to.Annotations[k] = v
		}
	}
}

func (ts *Transition) String() string {
	return string(ts.FromID) + `->` + string(ts.ToID)
}
