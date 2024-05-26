package memdriver

import (
	"fmt"

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

func (r *FlowRegistry) Do(cmd0 flowstate.Command) (*flowstate.StateCtx, error) {
	cmd, ok := cmd0.(*flowstate.GetFlowCommand)
	if !ok {
		return nil, flowstate.ErrCommandNotSupported
	}

	if cmd.StateCtx.Current.Transition.ToID == "" {
		return nil, fmt.Errorf("transition flow to is empty")
	}

	f, err := r.Flow(cmd.StateCtx.Current.Transition.ToID)
	if err != nil {
		return nil, err
	}

	cmd.Flow = f

	return nil, nil
}
