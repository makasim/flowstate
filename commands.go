package flowstate

import (
	"fmt"
	"strconv"
	"time"
)

type Command interface {
	cmd()
}

type command struct {
}

func (_ *command) cmd() {}

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

func Paused(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `paused`
}

func Pause(stateCtx *StateCtx) *PauseCommand {
	return &PauseCommand{
		StateCtx: stateCtx,
		FlowID:   stateCtx.Current.Transition.ToID,
	}
}

type PauseCommand struct {
	command

	StateCtx *StateCtx
	FlowID   FlowID
}

func (cmd *PauseCommand) WithTransit(fID FlowID) *PauseCommand {
	cmd.FlowID = fID
	return cmd
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
	command
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
	command
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
	command
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
	command
	StateCtx *StateCtx

	sync bool
}

func Noop(stateCtx *StateCtx) *NoopCommand {
	return &NoopCommand{
		StateCtx: stateCtx,
	}
}

type NoopCommand struct {
	command
	StateCtx *StateCtx
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

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommittableCommand interface {
	CommittableStateCtx() *StateCtx
}

type CommitCommand struct {
	command
	Commands []Command
}

func CommitStateCtx(stateCtx *StateCtx) *CommitStateCtxCommand {
	return &CommitStateCtxCommand{
		StateCtx: stateCtx,
	}
}

type CommitStateCtxCommand struct {
	command
	StateCtx *StateCtx
}

func (cmd *CommitStateCtxCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
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

func Serialize(serializableStateCtx, stateCtx *StateCtx, annotation string) *SerializeCommand {
	return &SerializeCommand{
		SerializableStateCtx: serializableStateCtx,
		StateCtx:             stateCtx,
		Annotation:           annotation,
	}

}

type SerializeCommand struct {
	command
	SerializableStateCtx *StateCtx
	StateCtx             *StateCtx
	Annotation           string
}

func Deserialize(stateCtx, deserializedStateCtx *StateCtx, annotation string) *DeserializeCommand {
	return &DeserializeCommand{
		StateCtx:             stateCtx,
		DeserializedStateCtx: deserializedStateCtx,
		Annotation:           annotation,
	}

}

type DeserializeCommand struct {
	command
	StateCtx             *StateCtx
	DeserializedStateCtx *StateCtx
	Annotation           string
}
