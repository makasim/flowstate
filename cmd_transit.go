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

func (cmd *TransitCommand) do() error {
	if cmd.FlowID == "" {
		return fmt.Errorf("flow id empty")
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		From:        cmd.StateCtx.Current.Transition.To,
		To:          cmd.FlowID,
		Annotations: nil,
	}

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
