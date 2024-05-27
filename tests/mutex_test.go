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
	var raceDetector int

	trkr := &tracker2{}

	fr := memdriver.NewFlowRegistry()
	fr.SetFlow("mutex", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		return nil, fmt.Errorf("must not be called; mutex is always paused")
	}))
	fr.SetFlow("lock", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		wCmd := flowstate.Watch(0, map[string]string{"mutex": "theName"})
		wCmd.SinceLatest = true

		if err := e.Do(wCmd); err != nil {
			return nil, err
		}
		w := wCmd.Listener
		defer w.Close()

		var mutexStateCtx *flowstate.StateCtx

		for {
			if mutexStateCtx != nil && mutexStateCtx.Current.Transition.ToID == "unlocked" {
				copyStateCtx := &flowstate.StateCtx{}
				stateCtx.CopyTo(copyStateCtx)

				copyMutexStateCtx := &flowstate.StateCtx{}
				mutexStateCtx.CopyTo(copyMutexStateCtx)

				conflictErr := &flowstate.ErrCommitConflict{}

				if err := e.Do(flowstate.Commit(
					flowstate.Pause(copyMutexStateCtx, `locked`),
					flowstate.Stack(copyMutexStateCtx, copyStateCtx),
					flowstate.Transit(copyStateCtx, `protected`),
				)); errors.As(err, conflictErr) {
					if conflictErr.Contains(mutexStateCtx.Current.ID) {
						mutexStateCtx = nil
						continue
					}

					return nil, err
				} else if err != nil {
					return nil, err
				}

				return flowstate.Execute(copyStateCtx), nil
			}

			select {
			case mutexStateCtx = <-w.Watch():
				continue
				// TODO: handle shutdown, timeout and other cases
			}

		}
	}))
	fr.SetFlow("protected", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		raceDetector += 1
		return flowstate.Transit(stateCtx, `unlock`), nil
	}))
	fr.SetFlow("unlock", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		mutexStateCtx := &flowstate.StateCtx{}

		if err := e.Do(flowstate.Commit(
			flowstate.Unstack(stateCtx, mutexStateCtx),
			flowstate.Pause(mutexStateCtx, `unlocked`),
			flowstate.End(stateCtx),
		)); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
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

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, fr)

	err := e.Do(flowstate.Commit(
		flowstate.Pause(mutexStateCtx, `unlocked`),
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

	time.Sleep(time.Millisecond * 500)

	visited := trkr.Visited()

	require.Len(t, visited, 9)
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
