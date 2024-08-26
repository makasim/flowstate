package flowstate

import (
	"fmt"
)

var StateAnnotation = `flowstate.state`

func Transit(stateCtx *StateCtx, fID FlowID) *TransitCommand {
	return &TransitCommand{
		StateCtx: stateCtx,
		FlowID:   fID,
	}
}

type TransitCommand struct {
	command
	StateCtx *StateCtx
	FlowID   FlowID
}

func (cmd *TransitCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

var DefaultTransitDoer DoerFunc = func(cmd0 Command) error {
	cmd, ok := cmd0.(*TransitCommand)
	if !ok {
		return ErrCommandNotSupported
	}

	if cmd.FlowID == "" {
		return fmt.Errorf("flow id empty")
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      cmd.StateCtx.Current.Transition.ToID,
		ToID:        cmd.FlowID,
		Annotations: nil,
	}

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
