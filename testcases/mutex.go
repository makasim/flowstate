package testcases

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Mutex(t TestingT, d flowstate.Driver, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	var raceDetector int

	trkr := &Tracker{}

	fr.SetFlow("mutex", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		return nil, fmt.Errorf("must not be called; mutex is always paused")
	}))
	fr.SetFlow("lock", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{"mutex": "theName"}).WithSinceLatest())
		defer w.Close()

		var mutexStateCtx *flowstate.StateCtx

		for {
			if mutexStateCtx != nil && mutexStateCtx.Current.Transition.To == "unlocked" {
				copyStateCtx := &flowstate.StateCtx{}
				stateCtx.CopyTo(copyStateCtx)

				copyMutexStateCtx := &flowstate.StateCtx{}
				mutexStateCtx.CopyTo(copyMutexStateCtx)

				if err := e.Do(flowstate.Commit(
					flowstate.Pause(copyMutexStateCtx).WithTransit(`locked`),
					flowstate.Serialize(copyMutexStateCtx, copyStateCtx, `mutex_state`),
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
				return flowstate.Noop(stateCtx), nil
			}

		}
	}))
	fr.SetFlow("protected", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		raceDetector += 1

		return flowstate.Transit(stateCtx, `unlock`), nil
	}))
	fr.SetFlow("unlock", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		mutexStateCtx := &flowstate.StateCtx{}
		if err := e.Do(flowstate.Commit(
			flowstate.Deserialize(stateCtx, mutexStateCtx, `mutex_state`),
			flowstate.Pause(mutexStateCtx).WithTransit(`unlocked`),
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

	l, _ := NewTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	err = e.Do(flowstate.Commit(
		flowstate.Pause(mutexStateCtx).WithTransit(`unlocked`),
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
