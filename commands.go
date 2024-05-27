package flowstate

import "time"

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

func Paused(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[StateAnnotation] == `paused`
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

func Resumed(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[StateAnnotation] == `resumed`
}

func Resume(stateCtx *StateCtx) *ResumeCommand {
	return &ResumeCommand{
		StateCtx: stateCtx,
	}
}

type ResumeCommand struct {
	StateCtx *StateCtx
}

func Ended(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[StateAnnotation] == `ended`
}

func End(stateCtx *StateCtx) *EndCommand {
	return &EndCommand{
		StateCtx: stateCtx,
	}
}

type EndCommand struct {
	StateCtx *StateCtx
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

type CommitCommand struct {
	Commands []Command
}

var DelayAtAnnotation = `flowstate.delay.at`
var DelayDurationAnnotation = `flowstate.delay.duration`

func Delayed(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[DelayAtAnnotation] != ``
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

func (cmd *DelayCommand) Prepare() error {
	delayedStateCtx := &StateCtx{}
	cmd.StateCtx.CopyTo(delayedStateCtx)

	delayedStateCtx.Transitions = append(delayedStateCtx.Transitions, delayedStateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      delayedStateCtx.Current.Transition.ToID,
		ToID:        delayedStateCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(DelayAtAnnotation, time.Now().Format(time.RFC3339Nano))
	nextTs.SetAnnotation(DelayDurationAnnotation, cmd.Duration.String())

	delayedStateCtx.Current.Transition = nextTs

	cmd.DelayStateCtx = delayedStateCtx

	return nil
}
