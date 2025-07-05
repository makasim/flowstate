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

func GetFlow(stateCtx *StateCtx) *GetFlowCommand {
	return &GetFlowCommand{
		StateCtx: stateCtx,
	}
}

type GetFlowCommand struct {
	command
	StateCtx *StateCtx

	// Result
	Flow Flow
}

func SetFlow(id FlowID, flow Flow) *SetFlowCommand {
	return &SetFlowCommand{
		FlowID: id,
		Flow:   flow,
	}
}

type SetFlowCommand struct {
	command
	FlowID FlowID
	Flow   Flow
}

type FlowRegistry struct {
	mux   sync.Mutex
	flows map[FlowID]Flow
}

func (fr *FlowRegistry) GetFlow(cmd *GetFlowCommand) error {
	fr.mux.Lock()
	defer fr.mux.Unlock()

	if cmd.StateCtx.Current.Transition.To == "" {
		return fmt.Errorf("transition flow to is empty")
	}

	if fr.flows == nil {
		return ErrFlowNotFound
	}

	f, ok := fr.flows[cmd.StateCtx.Current.Transition.To]
	if !ok {
		return ErrFlowNotFound
	}

	cmd.Flow = f
	return nil
}

func (fr *FlowRegistry) SetFlow(cmd *SetFlowCommand) error {
	if fr.flows == nil {
		fr.flows = make(map[FlowID]Flow)
	}

	fr.flows[cmd.FlowID] = cmd.Flow

	return nil
}
