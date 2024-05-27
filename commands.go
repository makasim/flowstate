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

var DeferAtAnnotation = `flowstate.defer.at`
var DeferDurationAnnotation = `flowstate.deferred.duration`

func Deferred(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[DeferAtAnnotation] != ``
}

func Defer(stateCtx *StateCtx, dur time.Duration) *DeferCommand {
	return &DeferCommand{
		OriginStateCtx: stateCtx,
		Duration:       dur,
	}

}

type DeferCommand struct {
	OriginStateCtx   *StateCtx
	DeferredStateCtx *StateCtx
	Duration         time.Duration
	Commit           bool
}

func (cmd *DeferCommand) Prepare() error {
	deferredStateCtx := &StateCtx{}
	cmd.OriginStateCtx.CopyTo(deferredStateCtx)

	deferredStateCtx.Transitions = append(deferredStateCtx.Transitions, deferredStateCtx.Current.Transition)

	nextTs := Transition{
		FromID:      deferredStateCtx.Current.Transition.ToID,
		ToID:        deferredStateCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(DeferAtAnnotation, time.Now().Format(time.RFC3339Nano))
	nextTs.SetAnnotation(DeferDurationAnnotation, cmd.Duration.String())

	deferredStateCtx.Current.Transition = nextTs

	cmd.DeferredStateCtx = deferredStateCtx

	return nil
}
