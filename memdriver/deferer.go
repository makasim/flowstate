package memdriver

import (
	"log"
	"time"

	"github.com/makasim/flowstate"
)

type Deferer struct {
	doer flowstate.Doer
}

func NewDeferer(d flowstate.Doer) *Deferer {
	return &Deferer{
		doer: d,
	}
}

func (d *Deferer) Do(cmd0 flowstate.Command) (*flowstate.StateCtx, error) {
	cmd, ok := cmd0.(*flowstate.DeferCommand)
	if !ok {
		return nil, flowstate.ErrCommandNotSupported
	}

	deferredStateCtx := &flowstate.StateCtx{}
	cmd.OriginStateCtx.CopyTo(deferredStateCtx)

	deferredStateCtx.Transitions = append(deferredStateCtx.Transitions, deferredStateCtx.Current.Transition)

	nextTs := flowstate.Transition{
		FromID:      deferredStateCtx.Current.Transition.ToID,
		ToID:        deferredStateCtx.Current.Transition.ToID,
		Annotations: nil,
	}
	nextTs.SetAnnotation(flowstate.DeferAtAnnotation, time.Now().Format(time.RFC3339Nano))
	nextTs.SetAnnotation(flowstate.DeferDurationAnnotation, cmd.Duration.String())

	deferredStateCtx.Current.Transition = nextTs

	cmd.DeferredStateCtx = deferredStateCtx

	// todo: replace naive implementation with real one
	go func() {
		t := time.NewTimer(cmd.Duration)
		defer t.Stop()

		<-t.C

		cmd.DeferredStateCtx.Doer = d.doer

		if _, err := d.doer.Do(flowstate.Execute(cmd.DeferredStateCtx)); err != nil {
			log.Printf(`ERROR: engine: defer: engine: execute: %s`, err)
		}
	}()

	return nil, nil
}
