package memdriver

import (
	"context"
	"log"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Delayer{}

type Delayer struct {
	e      *flowstate.Engine
	doneCh chan struct{}
}

func NewDelayer() *Delayer {
	return &Delayer{
		doneCh: make(chan struct{}),
	}
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

		select {
		case <-t.C:
			if err := d.e.Execute(cmd.DelayStateCtx); err != nil {
				log.Printf(`ERROR: memdriver: delayer: engine: execute: %s`, err)
			}
		case <-d.doneCh:
			log.Printf(`ERROR: memdriver: delayer: state delay was terminated`)
			return
		}
	}()

	return nil
}

func (d *Delayer) Init(e *flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Delayer) Shutdown(_ context.Context) error {
	close(d.doneCh)
	return nil
}
