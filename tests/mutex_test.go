package tests

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestMutex(t *testing.T) {
	pMutex := flowstate.Process{
		ID:  "mutexPID",
		Rev: 123,
		Nodes: []flowstate.Node{
			{
				ID:         "mutexNID",
				BehaviorID: "mutex",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:   "locked",
				ToID: "mutexNID",
			},
			{
				ID:   "unlocked",
				ToID: "mutexNID",
			},
		},
	}

	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "lockNID",
				BehaviorID: "lock",
			},
			{
				ID:         "protectedNID",
				BehaviorID: "protected",
			},
			{
				ID:         "unlockNID",
				BehaviorID: "unlock",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:   "lock",
				ToID: `lockNID`,
			},
			{
				ID:     `protected`,
				FromID: `lockNID`,
				ToID:   `protectedNID`,
			},
			{
				ID:     `unlock`,
				FromID: `protectedNID`,
				ToID:   `unlockNID`,
			},
		},
	}

	var raceDetector int

	trkr := &tracker{}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("mutex", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		return nil, fmt.Errorf("must not be called; mutex is always paused")
	}))
	br.SetBehavior("lock", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		e := taskCtx.Engine

		wCmd := flowstate.Watch(0, map[string]string{"mutex": "theName"})
		wCmd.SinceLatest = true

		if err := e.Do(wCmd); err != nil {
			return nil, err
		}
		w := wCmd.Watcher
		defer w.Close()

		var mutexTaskCtx *flowstate.TaskCtx

		for {
			if mutexTaskCtx != nil && mutexTaskCtx.Current.Transition.ID == "unlocked" {
				copyTaskCtx := &flowstate.TaskCtx{}
				taskCtx.CopyTo(copyTaskCtx)

				copyMutexTaskCtx := &flowstate.TaskCtx{}
				mutexTaskCtx.CopyTo(copyMutexTaskCtx)

				conflictErr := &flowstate.ErrCommitConflict{}

				if err := e.Do(flowstate.Commit(
					flowstate.Pause(copyMutexTaskCtx, `locked`),
					flowstate.Stack(copyMutexTaskCtx, copyTaskCtx),
					flowstate.Transit(copyTaskCtx, `protected`),
				)); errors.As(err, conflictErr) {
					if conflictErr.Contains(mutexTaskCtx.Current.ID) {
						mutexTaskCtx = nil
						continue
					}

					return nil, err
				} else if err != nil {
					return nil, err
				}

				return flowstate.Execute(copyTaskCtx), nil
			}

			select {
			case mutexTaskCtx = <-w.Watch():
				continue
				// TODO: handle shutdown, timeout and other cases
			}

		}
	}))
	br.SetBehavior("protected", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		raceDetector += 1
		return flowstate.Transit(taskCtx, `unlock`), nil
	}))
	br.SetBehavior("unlock", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) (flowstate.Command, error) {
		track(taskCtx, trkr)

		mutexTaskCtx := &flowstate.TaskCtx{}

		if err := taskCtx.Engine.Do(flowstate.Commit(
			flowstate.Unstack(taskCtx, mutexTaskCtx),
			flowstate.Pause(mutexTaskCtx, `unlocked`),
			flowstate.End(taskCtx),
		)); err != nil {
			return nil, err
		}

		return flowstate.Nop(taskCtx), nil
	}))

	mutexTaskCtx := &flowstate.TaskCtx{
		Current: flowstate.Task{
			ID:         "aMutexTID",
			Rev:        0,
			ProcessID:  pMutex.ID,
			ProcessRev: pMutex.Rev,
			Labels: map[string]string{
				"mutex": "theName",
			},
		},
		Process: pMutex,
	}

	var tasks []*flowstate.TaskCtx
	for i := 0; i < 3; i++ {
		taskCtx := &flowstate.TaskCtx{
			Current: flowstate.Task{
				ID:         flowstate.TaskID(fmt.Sprintf("aTID%d", i)),
				Rev:        0,
				ProcessID:  p.ID,
				ProcessRev: p.Rev,
			},
			Process: p,
		}
		tasks = append(tasks, taskCtx)
	}

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, br)

	err := e.Do(flowstate.Commit(
		flowstate.Pause(mutexTaskCtx, `unlocked`),
	))
	require.NoError(t, err)

	for _, taskCtx := range tasks {
		err = e.Do(flowstate.Commit(
			flowstate.Transit(taskCtx, `lock`),
			flowstate.Execute(taskCtx),
		))
		require.NoError(t, err)
	}

	time.Sleep(time.Millisecond * 500)

	//require.Equal(t, []flowstate.TransitionID{
	//	"lock",
	//	"lock",
	//	"lock",
	//	"protected",
	//	"unlock",
	//	"protected",
	//	"unlock",
	//	"protected",
	//	"unlock",
	//}, trkr.Visited())
}
