package memdriver

import (
	"github.com/makasim/flowstate"
)

type FlowRegistry struct {
	flows map[flowstate.FlowID]flowstate.Flow
}

func NewFlowRegistry() *FlowRegistry {
	return &FlowRegistry{}
}

func (r *FlowRegistry) SetFlow(id flowstate.FlowID, f flowstate.Flow) {
	if r.flows == nil {
		r.flows = make(map[flowstate.FlowID]flowstate.Flow)
	}

	r.flows[id] = f
}

func (r *FlowRegistry) Flow(id flowstate.FlowID) (flowstate.Flow, error) {
	if r.flows == nil {
		return nil, flowstate.ErrFlowNotFound
	}

	f, ok := r.flows[id]
	if !ok {
		return nil, flowstate.ErrFlowNotFound
	}

	return f, nil
}
