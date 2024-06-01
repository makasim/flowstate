package memdriver

import (
	"fmt"

	"github.com/makasim/flowstate"
)

type FlowRegistry struct {
	flows map[flowstate.FlowID]flowstate.Flow
}

func (fr *FlowRegistry) SetFlow(id flowstate.FlowID, f flowstate.Flow) {
	if fr.flows == nil {
		fr.flows = make(map[flowstate.FlowID]flowstate.Flow)
	}

	fr.flows[id] = f
}

func (fr *FlowRegistry) Flow(id flowstate.FlowID) (flowstate.Flow, error) {
	if fr.flows == nil {
		return nil, flowstate.ErrFlowNotFound
	}

	f, ok := fr.flows[id]
	if !ok {
		return nil, flowstate.ErrFlowNotFound
	}

	return f, nil
}

func NewFlowGetter(fr *FlowRegistry) flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*flowstate.GetFlowCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

		if cmd.StateCtx.Current.Transition.ToID == "" {
			return fmt.Errorf("transition flow to is empty")
		}

		f, err := fr.Flow(cmd.StateCtx.Current.Transition.ToID)
		if err != nil {
			return err
		}

		cmd.Flow = f
		return nil
	})
}
