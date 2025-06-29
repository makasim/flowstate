package flowstate

import "fmt"

type FlowID string

type Flow interface {
	Execute(stateCtx *StateCtx, e Engine) (Command, error)
}

type FlowFunc func(stateCtx *StateCtx, e Engine) (Command, error)

func (f FlowFunc) Execute(stateCtx *StateCtx, e Engine) (Command, error) {
	return f(stateCtx, e)
}

type FlowRegistry struct {
	flows map[FlowID]Flow
}

func (fr *FlowRegistry) SetFlow(id FlowID, f Flow) {
	if fr.flows == nil {
		fr.flows = make(map[FlowID]Flow)
	}

	fr.flows[id] = f
}

func (fr *FlowRegistry) Flow(id FlowID) (Flow, error) {
	if fr.flows == nil {
		return nil, ErrFlowNotFound
	}

	f, ok := fr.flows[id]
	if !ok {
		return nil, ErrFlowNotFound
	}

	return f, nil
}

func (fr *FlowRegistry) Do(cmd *GetFlowCommand) error {
	if cmd.StateCtx.Current.Transition.To == "" {
		return fmt.Errorf("transition flow to is empty")
	}

	f, err := fr.Flow(cmd.StateCtx.Current.Transition.To)
	if err != nil {
		return err
	}

	cmd.Flow = f
	return nil
}
