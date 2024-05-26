package flowstate

import (
	"time"
)

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

// A driver must implement a command doer.
