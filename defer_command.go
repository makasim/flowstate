package flowstate

import (
	"time"
)

var deferAtAnnotation = `flowstate.defer.at`
var deferDurationAnnotation = `flowstate.deferred.duration`

func Deferred(taskCtx *TaskCtx) bool {
	return taskCtx.Current.Transition.Annotations[deferAtAnnotation] != ``
}

func Defer(taskCtx *TaskCtx, dur time.Duration) *DeferCommand {
	return &DeferCommand{
		OriginTaskCtx: taskCtx,
		Duration:      dur,
	}

}

type DeferCommand struct {
	OriginTaskCtx   *TaskCtx
	DeferredTaskCtx *TaskCtx
	Duration        time.Duration
	Commit          bool
}

func (cmd *DeferCommand) Prepare() error {
	deferredTaskCtx := &TaskCtx{}
	cmd.OriginTaskCtx.CopyTo(deferredTaskCtx)

	deferredTaskCtx.Transitions = append(deferredTaskCtx.Transitions, deferredTaskCtx.Current.Transition)

	nextTs := Transition{
		FromID:      deferredTaskCtx.Current.Transition.ToID,
		ToID:        deferredTaskCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(deferAtAnnotation, time.Now().Format(time.RFC3339Nano))
	nextTs.SetAnnotation(deferDurationAnnotation, cmd.Duration.String())

	deferredTaskCtx.Current.Transition = nextTs

	cmd.DeferredTaskCtx = deferredTaskCtx

	return nil
}
