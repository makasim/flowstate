package memdriver

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Delayer{}

type Delayer struct {
	e      flowstate.Engine
	stopCh chan struct{}
	wg     sync.WaitGroup
	l      *slog.Logger
}

func NewDelayer(l *slog.Logger) *Delayer {
	return &Delayer{
		stopCh: make(chan struct{}),
		l:      l,
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
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()

		t := time.NewTimer(cmd.Duration)
		defer t.Stop()

		stateCtx := cmd.DelayStateCtx

		select {
		case <-t.C:
			if stateCtx.Current.Transition.Annotations[flowstate.DelayCommitAnnotation] == `true` {
				if err := d.e.Do(flowstate.Commit(
					flowstate.CommitStateCtx(cmd.DelayStateCtx),
				)); flowstate.IsErrRevMismatch(err) {
					d.l.Info("delayer: commit conflict",
						"sess", cmd.SessID(),
						"conflict", err.Error(),
						"id", stateCtx.Current.ID,
						"rev", stateCtx.Current.Rev,
					)
					return
				} else if err != nil {
					d.l.Error("delayer: commit failed",
						"sess", cmd.SessID(),
						"error", err,
						"id", stateCtx.Current.ID,
						"rev", stateCtx.Current.Rev,
					)
					return
				}
			}

			if err := d.e.Execute(stateCtx); err != nil {
				d.l.Error("delayer: execute failed",
					"sess", stateCtx.SessID(),
					"error", err,
					"id", stateCtx.Current.ID,
					"rev", stateCtx.Current.Rev,
				)
			}
		case <-d.stopCh:
			d.l.Error("delayer: delaying terminated",
				"id", stateCtx.Current.ID,
				"rev", stateCtx.Current.Rev,
			)
			return
		}
	}()

	return nil
}

func (d *Delayer) Init(e flowstate.Engine) error {
	d.e = e
	d.wg.Add(1)
	return nil
}

func (d *Delayer) Shutdown(ctx context.Context) error {
	select {
	case <-d.stopCh:
		return fmt.Errorf(`already stopped`)
	default:
	}

	close(d.stopCh)
	d.wg.Done()

	stoppedCh := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(stoppedCh)
	}()

	select {
	case <-stoppedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
