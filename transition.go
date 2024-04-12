package flowstate

type TransitionID string

type Transition struct {
	ID          TransitionID      `json:"id"`
	FromID      NodeID            `json:"from"`
	ToID        NodeID            `json:"to"`
	Annotations map[string]string `json:"annotations"`
}

func (ts *Transition) SetAnnotation(name, value string) {
	if ts.Annotations == nil {
		ts.Annotations = make(map[string]string)
	}
	ts.Annotations[name] = value
}
