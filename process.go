package flowstate

import "fmt"

var ErrNodeNotFound = fmt.Errorf("node not found")
var ErrTransitionNotFound = fmt.Errorf("transition not found")

type ProcessID string

type Process struct {
	ID  ProcessID `json:"id"`
	Rev int64     `json:"rev"`

	Nodes       []Node       `json:"nodes"`
	Transitions []Transition `json:"transitions"`
}

func (p *Process) Transition(id TransitionID) (Transition, error) {
	if id == `` {
		return Transition{}, fmt.Errorf("transition id empty")
	}

	for i := range p.Transitions {
		if p.Transitions[i].ID == id {
			return p.Transitions[i], nil
		}
	}

	return Transition{}, fmt.Errorf("%w; ts: %s", ErrTransitionNotFound, id)
}

func (p *Process) Node(id NodeID) (Node, error) {
	if id == `` {
		return Node{}, fmt.Errorf("node id empty")
	}

	for i := range p.Nodes {
		if p.Nodes[i].ID == id {
			return p.Nodes[i], nil
		}
	}

	return Node{}, fmt.Errorf("%w; node: %s", ErrNodeNotFound, id)
}
