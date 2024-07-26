package flowstate

import (
	"fmt"
	"time"
)

var DelayAtAnnotation = `flowstate.delay.at`
var DelayDurationAnnotation = `flowstate.delay.duration`
var DelayCommitAnnotation = `flowstate.delay.commit`

func Delayed(state State) bool {
	return state.Transition.Annotations[DelayAtAnnotation] != ``
}

func Delay(stateCtx *StateCtx, dur time.Duration) *DelayCommand {
	return &DelayCommand{
		StateCtx: stateCtx,
		Duration: dur,
	}

}

type DelayCommand struct {
	command
	StateCtx      *StateCtx
	DelayStateCtx *StateCtx
	Duration      time.Duration
	Commit        bool
}

func (cmd *DelayCommand) WithCommit(commit bool) *DelayCommand {
	cmd.Commit = commit
	return cmd
}

func (cmd *DelayCommand) Prepare() error {
	delayedStateCtx := cmd.StateCtx.CopyTo(&StateCtx{})

	delayedStateCtx.Transitions = append(delayedStateCtx.Transitions, delayedStateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      delayedStateCtx.Current.Transition.ToID,
		ToID:        delayedStateCtx.Current.Transition.ToID,
		Annotations: nil,
	}

	if Paused(delayedStateCtx.Current) {
		nextTs.SetAnnotation(StateAnnotation, `resumed`)
	}

	nextTs.SetAnnotation(DelayAtAnnotation, time.Now().Format(time.RFC3339Nano))
	nextTs.SetAnnotation(DelayDurationAnnotation, cmd.Duration.String())
	nextTs.SetAnnotation(DelayCommitAnnotation, fmt.Sprintf("%v", cmd.Commit))

	delayedStateCtx.Current.Transition = nextTs

	cmd.DelayStateCtx = delayedStateCtx

	return nil
}
