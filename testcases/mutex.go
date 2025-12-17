package testcases

import (
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Mutex(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	var raceDetector int

	trkr := &Tracker{}

	mustSetFlow(fr, "mutex", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		return nil, fmt.Errorf("must not be called; mutex is always paused")
	}))
	mustSetFlow(fr, "lock", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{"mutex": "theName"}).WithSinceLatest())
		defer w.Close()

		var mutexStateCtx *flowstate.StateCtx

		for {
			if mutexStateCtx != nil && mutexStateCtx.Current.Annotation(`state`) == "unlocked" {
				copyStateCtx := &flowstate.StateCtx{}
				stateCtx.CopyTo(copyStateCtx)

				copyMutexStateCtx := &flowstate.StateCtx{}
				mutexStateCtx.CopyTo(copyMutexStateCtx)

				if err := e.Do(flowstate.Commit(
					flowstate.Park(copyMutexStateCtx).WithAnnotation(`state`, `locked`),
					flowstate.Stack(copyStateCtx, copyMutexStateCtx, `mutex_state`),
					flowstate.Transit(copyStateCtx, `protected`),
				)); flowstate.IsErrRevMismatchContains(err, mutexStateCtx.Current.ID) {
					mutexStateCtx = nil
					continue
				} else if flowstate.IsErrRevMismatch(err) {
					return nil, err
				} else if err != nil {
					return nil, err
				}

				return flowstate.Execute(copyStateCtx), nil
			}

			select {
			case mutexState := <-w.Next():
				if mutexStateCtx == nil {
					mutexStateCtx = &flowstate.StateCtx{}
				}

				mutexStateCtx = mutexState.CopyToCtx(mutexStateCtx)

				continue
			case <-stateCtx.Done():
				return flowstate.Noop(), nil
			}

		}
	}))
	mustSetFlow(fr, "protected", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		raceDetector += 1

		return flowstate.Transit(stateCtx, `unlock`), nil
	}))
	mustSetFlow(fr, "unlock", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		mutexStateCtx := &flowstate.StateCtx{}
		if err := e.Do(flowstate.Commit(
			flowstate.Unstack(stateCtx, mutexStateCtx, `mutex_state`),
			flowstate.Park(mutexStateCtx).WithAnnotation(`state`, `unlocked`),
			flowstate.Park(stateCtx),
		)); err != nil {
			return nil, err
		}

		return flowstate.Noop(), nil
	}))

	mutexStateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aMutexTID",
			Labels: map[string]string{
				"mutex": "theName",
			},
		},
	}

	var states []*flowstate.StateCtx
	for i := 0; i < 3; i++ {
		statesCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: flowstate.StateID(fmt.Sprintf("aTID%d", i)),
			},
		}
		states = append(states, statesCtx)
	}

	err := e.Do(flowstate.Commit(
		flowstate.Park(mutexStateCtx).WithAnnotation(`state`, `unlocked`),
	))
	require.NoError(t, err)

	for _, stateCtx := range states {
		err = e.Do(
			flowstate.Commit(
				flowstate.Transit(stateCtx, `lock`),
			),
			flowstate.Execute(stateCtx),
		)
		require.NoError(t, err)
	}

	visited := trkr.WaitVisitedCountGreaterOrEqual(t, 9, time.Millisecond*1200)
	var lockCnt int
	var protectedCnt int
	var unlockCnt int
	for i := range visited {
		switch visited[i] {
		case "lock":
			lockCnt++
		case "protected":
			protectedCnt++
			require.Equal(t, visited[i+1], "unlock")
		case "unlock":
			unlockCnt++
		}
	}
	require.Equal(t, 3, lockCnt)
	require.Equal(t, 3, protectedCnt)
	require.Equal(t, 3, unlockCnt)
}
