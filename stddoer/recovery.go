package stddoer

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/makasim/flowstate"
)

type RecoveryDoer struct {
	failoverDur time.Duration

	w      flowstate.Watcher
	e      *flowstate.Engine
	doneCh chan struct{}
	log    []flowstate.State
}

func Recovery(failoverDur time.Duration) flowstate.Doer {
	return &RecoveryDoer{
		doneCh:      make(chan struct{}),
		failoverDur: failoverDur,
	}
}

func (d *RecoveryDoer) Do(cmd0 flowstate.Command) error {
	if _, ok := cmd0.(*recoveryCommit); ok {
		return nil
	}

	return flowstate.ErrCommandNotSupported
}

func (d *RecoveryDoer) Init(e *flowstate.Engine) error {

	w, err := e.Watch(0, nil)
	if err != nil {
		return err
	}

	d.w = w
	d.e = e

	t := time.NewTicker(d.failoverDur)

	go func() {
		for {
			select {
			case <-d.doneCh:
				return
			case state0 := <-w.Watch():
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

func (d *RecoveryDoer) checkLog() error {

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
			flowstate.Commit(&recoveryCommit{stateCtx: stateCtx}),
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

func (d *RecoveryDoer) Shutdown(_ context.Context) error {
	d.w.Close()
	close(d.doneCh)

	return nil
}

type recoveryCommit struct {
	stateCtx *flowstate.StateCtx
}

func (cmd *recoveryCommit) CommittableStateCtx() *flowstate.StateCtx {
	return cmd.stateCtx
}

func (cmd *recoveryCommit) Prepare() error {
	return nil
}
