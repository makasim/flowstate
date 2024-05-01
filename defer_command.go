package flowstate

import (
	"time"
)

func Deferred(taskCtx *TaskCtx) bool {
	return taskCtx.Current.Transition.Annotations[deferAtAnnotation] != ``
}

func Defer(taskCtx *TaskCtx, dur time.Duration) *DeferCommand {
	return &DeferCommand{
		OriginTaskCtx: taskCtx,
		Duration:      dur,
	}

}

var deferAtAnnotation = `flowstate.defer.at`
var deferDurationAnnotation = `flowstate.deferred.duration`

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

	deferredTaskCtx.Current.Transition.SetAnnotation(deferAtAnnotation, time.Now().Format(time.RFC3339Nano))
	deferredTaskCtx.Current.Transition.SetAnnotation(deferDurationAnnotation, cmd.Duration.String())

	cmd.DeferredTaskCtx = deferredTaskCtx

	return nil
}
