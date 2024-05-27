package memdriver

import (
	"context"
	"log"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Delayer{}

type Delayer struct {
	e *flowstate.Engine
}

func NewDelayer() *Delayer {
	return &Delayer{}
}

func (d *Delayer) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*flowstate.DelayCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if err := cmd.Prepare(); err != nil {
		return err
	}

	// todo: replace naive implementation with real one
	go func() {
		t := time.NewTimer(cmd.Duration)
		defer t.Stop()

		<-t.C

		if err := d.e.Execute(cmd.DelayStateCtx); err != nil {
			log.Printf(`ERROR: engine: defer: engine: execute: %s`, err)
		}
	}()

	return nil
}

func (d *Delayer) Init(e *flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Delayer) Shutdown(_ context.Context) error {
	return nil
}
