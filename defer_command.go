package flowstate

import (
	"time"
)

var deferAtAnnotation = `flowstate.defer.at`
var deferDurationAnnotation = `flowstate.deferred.duration`

func Deferred(stateCtx *StateCtx) bool {
	return stateCtx.Current.Transition.Annotations[deferAtAnnotation] != ``
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
	nextTs.SetAnnotation(deferAtAnnotation, time.Now().Format(time.RFC3339Nano))
	nextTs.SetAnnotation(deferDurationAnnotation, cmd.Duration.String())

	deferredStateCtx.Current.Transition = nextTs

	cmd.DeferredStateCtx = deferredStateCtx

	return nil
}
