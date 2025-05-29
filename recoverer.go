package flowstate

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"
)

type RecovererDoer struct {
	failoverDur time.Duration

	e         Engine
	doneCh    chan struct{}
	stoppedCh chan struct{}
	log       []State
}

func Recoverer(failoverDur time.Duration) Doer {
	return &RecovererDoer{
		failoverDur: failoverDur,
		doneCh:      make(chan struct{}),
		stoppedCh:   make(chan struct{}),
	}
}

func (d *RecovererDoer) Do(_ Command) error {
	return ErrCommandNotSupported
}

func (d *RecovererDoer) Init(e Engine) error {
	d.e = e

	go func() {
		defer close(d.stoppedCh)

		t := time.NewTicker(d.failoverDur)
		defer t.Stop()

		w := NewWatcher(e, &GetManyCommand{
			SinceRev: 0,
			Labels:   nil,
		})
		defer w.Close()

		for {
			select {
			case <-d.doneCh:
				return
			case state0 := <-w.Next():
				state := state0.CopyTo(&State{})
				d.log = append(d.log, state)
			case <-t.C:
				if err := d.checkLog(); err != nil {
					log.Printf("ERROR: recovery: check log: %v", err)
				}
			}
		}
	}()

	return nil
}

func (d *RecovererDoer) checkLog() error {

	visited := make(map[StateID]struct{})

	failoverTime := time.Now().Add(-d.failoverDur)

	newLog := make([]State, 0, len(d.log))
	for i := len(d.log) - 1; i >= 0; i-- {
		state := d.log[i]

		if Paused(state) || Ended(state) {
			continue
		} else if _, ok := visited[state.ID]; ok {
			continue
		}

		if !state.CommittedAt().Before(failoverTime) {
			visited[state.ID] = struct{}{}
			newLog = append(newLog, state)
			continue
		}

		conflictErr := &ErrRevMismatch{}

		recoveryAttempt := RecoveryAttempt(state)
		if recoveryAttempt >= 2 {
			stateCtx := state.CopyToCtx(&StateCtx{})
			if err := d.e.Do(
				Commit(End(stateCtx)),
			); errors.As(err, conflictErr) {
				visited[state.ID] = struct{}{}
				continue
			} else if err != nil {
				return err
			}

			visited[state.ID] = struct{}{}
			continue
		}

		stateCtx := state.CopyToCtx(&StateCtx{})
		stateCtx.Current.Transition.SetAnnotation(RecoveryAttemptAnnotation, strconv.Itoa(recoveryAttempt+1))
		if err := d.e.Do(
			Commit(CommitStateCtx(stateCtx)),
			Execute(stateCtx),
		); errors.As(err, conflictErr) {
			visited[state.ID] = struct{}{}
			continue
		} else if err != nil {
			return err
		}

		visited[state.ID] = struct{}{}
		continue
	}

	d.log = newLog
	return nil
}

func (d *RecovererDoer) Shutdown(ctx context.Context) error {
	close(d.doneCh)

	select {
	case <-d.stoppedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
