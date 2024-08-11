package stddoer

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/makasim/flowstate"
)

type RecovererDoer struct {
	failoverDur time.Duration

	wl        flowstate.WatchListener
	e         *flowstate.Engine
	doneCh    chan struct{}
	stoppedCh chan struct{}
	log       []flowstate.State
}

func Recoverer(failoverDur time.Duration) flowstate.Doer {
	return &RecovererDoer{
		failoverDur: failoverDur,
		doneCh:      make(chan struct{}),
		stoppedCh:   make(chan struct{}),
	}
}

func (d *RecovererDoer) Do(_ flowstate.Command) error {
	return flowstate.ErrCommandNotSupported
}

func (d *RecovererDoer) Init(e *flowstate.Engine) error {
	cmd := flowstate.Watch(nil)
	if err := e.Do(cmd); err != nil {
		return err
	}

	d.wl = cmd.Listener
	d.e = e

	go func() {
		defer close(d.stoppedCh)

		t := time.NewTicker(d.failoverDur)
		defer t.Stop()

		for {
			select {
			case <-d.doneCh:
				return
			case state0 := <-d.wl.Listen():
				state := state0.CopyTo(&flowstate.State{})
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

	visited := make(map[flowstate.StateID]struct{})

	failoverTime := time.Now().Add(-d.failoverDur)

	newLog := make([]flowstate.State, 0, len(d.log))
	for i := len(d.log) - 1; i >= 0; i-- {
		state := d.log[i]

		if flowstate.Paused(state) || flowstate.Ended(state) {
			continue
		} else if _, ok := visited[state.ID]; ok {
			continue
		}

		if !state.CommittedAt().Before(failoverTime) {
			visited[state.ID] = struct{}{}
			newLog = append(newLog, state)
			continue
		}

		conflictErr := &flowstate.ErrCommitConflict{}

		recoveryAttempt := flowstate.RecoveryAttempt(state)
		if recoveryAttempt >= 2 {
			stateCtx := state.CopyToCtx(&flowstate.StateCtx{})
			if err := d.e.Do(
				flowstate.Commit(flowstate.End(stateCtx)),
			); errors.As(err, conflictErr) {
				visited[state.ID] = struct{}{}
				continue
			} else if err != nil {
				return err
			}

			visited[state.ID] = struct{}{}
			continue
		}

		stateCtx := state.CopyToCtx(&flowstate.StateCtx{})
		stateCtx.Current.Transition.SetAnnotation(flowstate.RecoveryAttemptAnnotation, strconv.Itoa(recoveryAttempt+1))
		if err := d.e.Do(
			flowstate.Commit(flowstate.CommitStateCtx(stateCtx)),
			flowstate.Execute(stateCtx),
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
	d.wl.Close()
	close(d.doneCh)

	select {
	case <-d.stoppedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
