package flowstate

import (
	"time"
)

func Defer(taskCtx *TaskCtx, dur time.Duration, commit bool) *DeferCommand {
	return &DeferCommand{
		OriginTaskCtx: taskCtx,
		Duration:      dur,
		Commit:        commit,
	}

}

var DeferAtAnnotation = `flowstate.defer.at`
var DeferDurationAnnotation = `flowstate.deferred.duration`

type DeferCommand struct {
	OriginTaskCtx   *TaskCtx
	DeferredTaskCtx *TaskCtx
	Duration        time.Duration
	Commit          bool
}

func (cmd *DeferCommand) Prepare() error {
	deferredTaskCtx := &TaskCtx{}
	cmd.OriginTaskCtx.CopyTo(deferredTaskCtx)

	deferredTaskCtx.Engine = nil

	if err := Transit(deferredTaskCtx, deferredTaskCtx.Current.Transition.ID).Prepare(); err != nil {
		return err
	}

	deferredTaskCtx.Current.Transition.SetAnnotation(DeferAtAnnotation, time.Now().Format(time.RFC3339Nano))
	deferredTaskCtx.Current.Transition.SetAnnotation(DeferDurationAnnotation, cmd.Duration.String())

	cmd.DeferredTaskCtx = deferredTaskCtx

	return nil
}
