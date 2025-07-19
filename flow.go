package flowstate

import (
	"fmt"
	"sync"
)

type Flow interface {
	Execute(stateCtx *StateCtx, e Engine) (Command, error)
}

type FlowFunc func(stateCtx *StateCtx, e Engine) (Command, error)

func (f FlowFunc) Execute(stateCtx *StateCtx, e Engine) (Command, error) {
	return f(stateCtx, e)
}

type FlowRegistry interface {
	Flow(id TransitionID) (Flow, error)
	SetFlow(id TransitionID, flow Flow) error
	UnsetFlow(id TransitionID) error
}

var _ FlowRegistry = (*DefaultFlowRegistry)(nil)

type DefaultFlowRegistry struct {
	mux   sync.Mutex
	flows map[TransitionID]Flow
}

func (fr *DefaultFlowRegistry) Flow(id TransitionID) (Flow, error) {
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

func (fr *DefaultFlowRegistry) SetFlow(id TransitionID, flow Flow) error {
	if id == "" {
		return fmt.Errorf("flow id empty")
	}

	fr.mux.Lock()
	defer fr.mux.Unlock()

	if fr.flows == nil {
		fr.flows = make(map[TransitionID]Flow)
	}

	fr.flows[id] = flow
	return nil
}

func (fr *DefaultFlowRegistry) UnsetFlow(id TransitionID) error {
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
