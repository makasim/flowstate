package memdriver

import (
	"context"
	"log"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Deferer{}

type Deferer struct {
	e flowstate.Engine
}

func NewDeferer() *Deferer {
	return &Deferer{}
}

func (d *Deferer) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*flowstate.DeferCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
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

		if err := d.e.Execute(cmd.DeferredStateCtx); err != nil {
			log.Printf(`ERROR: engine: defer: engine: execute: %s`, err)
		}
	}()

	return nil
}

func (d *Deferer) Init(e flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Deferer) Shutdown(ctx context.Context) error {
	return nil
}
