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

type FlowRegistry interface {
	Flow(id FlowID) (Flow, error)
	SetFlow(id FlowID, flow Flow) error
	UnsetFlow(id FlowID) error
}

var _ FlowRegistry = (*DefaultFlowRegistry)(nil)

type DefaultFlowRegistry struct {
	mux   sync.Mutex
	flows map[FlowID]Flow
}

func (fr *DefaultFlowRegistry) Flow(id FlowID) (Flow, error) {
	if id == "" {
		return nil, fmt.Errorf("flow id empty")
	}

	fr.mux.Lock()
	defer fr.mux.Unlock()

	if fr.flows == nil {
		return nil, ErrFlowNotFound
	}

	f, ok := fr.flows[id]
	if !ok {
		return nil, ErrFlowNotFound
	}

	return f, nil
}

func (fr *DefaultFlowRegistry) SetFlow(id FlowID, flow Flow) error {
	if id == "" {
		return fmt.Errorf("flow id empty")
	}

	fr.mux.Lock()
	defer fr.mux.Unlock()

	if fr.flows == nil {
		fr.flows = make(map[FlowID]Flow)
	}

	fr.flows[id] = flow
	return nil
}

func (fr *DefaultFlowRegistry) UnsetFlow(id FlowID) error {
	if id == "" {
		return fmt.Errorf("flow id empty")
	}

	fr.mux.Lock()
	defer fr.mux.Unlock()

	if fr.flows == nil {
		return nil
	}

	delete(fr.flows, id)
	return nil
}
