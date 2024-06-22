package flowstate

import (
	"fmt"
	"strconv"
	"time"
)

var StateAnnotation = `flowstate.state`

func Transit(stateCtx *StateCtx, fID FlowID) *TransitCommand {
	return &TransitCommand{
		StateCtx: stateCtx,
		FlowID:   fID,
	}
}

type TransitCommand struct {
	StateCtx *StateCtx
	FlowID   FlowID
}

func (cmd *TransitCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func Paused(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `paused`
}

func Pause(stateCtx *StateCtx, fID FlowID) *PauseCommand {
	return &PauseCommand{
		StateCtx: stateCtx,
		FlowID:   fID,
	}
}

type PauseCommand struct {
	StateCtx *StateCtx
	FlowID   FlowID
}

func (cmd *PauseCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func Resumed(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `resumed`
}

func Resume(stateCtx *StateCtx) *ResumeCommand {
	return &ResumeCommand{
		StateCtx: stateCtx,
	}
}

type ResumeCommand struct {
	StateCtx *StateCtx
}

func (cmd *ResumeCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func Ended(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `ended`
}

func End(stateCtx *StateCtx) *EndCommand {
	return &EndCommand{
		StateCtx: stateCtx,
	}
}

type EndCommand struct {
	StateCtx *StateCtx
}

func (cmd *EndCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func GetWatcher(sinceRev int64, labels map[string]string) *GetWatcherCommand {
	return &GetWatcherCommand{
		SinceRev: sinceRev,
		Labels:   labels,
	}
}

type GetWatcherCommand struct {
	SinceRev    int64
	SinceLatest bool
	Labels      map[string]string

	Watcher Watcher
}

func Execute(stateCtx *StateCtx) *ExecuteCommand {
	return &ExecuteCommand{
		StateCtx: stateCtx,
	}
}

type ExecuteCommand struct {
	StateCtx *StateCtx

	sync bool
}

func Noop(stateCtx *StateCtx) *NoopCommand {
	return &NoopCommand{
		StateCtx: stateCtx,
	}
}

type NoopCommand struct {
	StateCtx *StateCtx
}

func GetFlow(stateCtx *StateCtx) *GetFlowCommand {
	return &GetFlowCommand{
		StateCtx: stateCtx,
	}
}

type GetFlowCommand struct {
	StateCtx *StateCtx

	// Result
	Flow Flow
}

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommittableCommand interface {
	CommittableStateCtx() *StateCtx
}

type CommitCommand struct {
	Commands []Command
}

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
	StateCtx      *StateCtx
	DelayStateCtx *StateCtx
	Duration      time.Duration
	Commit        bool
}

func (cmd *DelayCommand) CommittableStateCtx() *StateCtx {
	return cmd.DelayStateCtx
}

func (cmd *DelayCommand) Prepare() error {
	delayedStateCtx := cmd.StateCtx.CopyTo(&StateCtx{})

	delayedStateCtx.Transitions = append(delayedStateCtx.Transitions, delayedStateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      delayedStateCtx.Current.Transition.ToID,
		ToID:        delayedStateCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(DelayAtAnnotation, time.Now().Format(time.RFC3339Nano))
	nextTs.SetAnnotation(DelayDurationAnnotation, cmd.Duration.String())
	nextTs.SetAnnotation(DelayCommitAnnotation, fmt.Sprintf("%v", cmd.Commit))

	delayedStateCtx.Current.Transition = nextTs

	cmd.DelayStateCtx = delayedStateCtx

	return nil
}

var RecoveryAttemptAnnotation = `flowstate.recovery_attempt`

func RecoveryAttempt(state State) int {
	attempt, _ := strconv.Atoi(state.Transition.Annotations[RecoveryAttemptAnnotation])
	return attempt
}
