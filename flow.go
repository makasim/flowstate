package flowstate

import (
	"fmt"
	"sync"
)

type FlowID string

type Flow interface {
	Execute(stateCtx *StateCtx, e Engine) (Command, error)
}

type FlowFunc func(stateCtx *StateCtx, e Engine) (Command, error)

func (f FlowFunc) Execute(stateCtx *StateCtx, e Engine) (Command, error) {
	return f(stateCtx, e)
}

type FlowRegistry struct {
	mux   sync.Mutex
	flows map[FlowID]Flow
}

func (fr *FlowRegistry) Flow(id FlowID) (Flow, error) {
	fr.mux.Lock()
	defer fr.mux.Unlock()

	if id == "" {
		return nil, fmt.Errorf("flow id empty")
	}

	if fr.flows == nil {
		return nil, ErrFlowNotFound
	}

	f, ok := fr.flows[id]
	if !ok {
		return nil, ErrFlowNotFound
	}

	return f, nil
}

func (fr *FlowRegistry) SetFlow(id FlowID, flow Flow) error {
	fr.mux.Lock()
	defer fr.mux.Unlock()

	if fr.flows == nil {
		fr.flows = make(map[FlowID]Flow)
	}

	fr.flows[id] = flow
	return nil
}
