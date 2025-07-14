package flowstate

import (
	"fmt"
)

var StateAnnotation = `flowstate.state`

func Transit(stateCtx *StateCtx, to TransitionID) *TransitCommand {
	return &TransitCommand{
		StateCtx: stateCtx,
		To:       to,
	}
}

type TransitCommand struct {
	command
	StateCtx *StateCtx
	To       TransitionID
}

func (cmd *TransitCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func (cmd *TransitCommand) Do() error {
	if cmd.To == "" {
		return fmt.Errorf("flow id empty")
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{
		From:        cmd.StateCtx.Current.Transition.To,
		To:          cmd.To,
		Annotations: nil,
	}

	cmd.StateCtx.Current.Transition = nextTs

	return nil
}
