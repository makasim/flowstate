package testcases

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Actor(t TestingT, d flowstate.Driver, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{IncludeTaskID: true}

	fr.SetFlow("actor", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{
			"actor.foo": "inbox",
		}))
		defer w.Close()

		t := time.NewTimer(time.Millisecond * 100)
		for {
			select {
			case msgState := <-w.Next():
				Track(msgState.CopyToCtx(&flowstate.StateCtx{}), trkr)
				// do stuff here
			case <-t.C:
				// make sure recovery is not triggered, t.C must be less than failoverDur
				if err := e.Do(flowstate.Commit(
					flowstate.Transit(stateCtx, `actor`),
				)); err != nil {
					return nil, err
				}
			case <-stateCtx.Done():
				return flowstate.Commit(
					flowstate.Pause(stateCtx),
				), nil
			}
		}
	}))
	fr.SetFlow("inbox", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		return nil, fmt.Errorf("must never be executed")
	}))

	l, _ := NewTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	actorStateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "actorTID",
		},
	}

	require.NoError(t, e.Do(
		flowstate.Commit(
			flowstate.Transit(actorStateCtx, `actor`),
		),
		flowstate.Execute(actorStateCtx),
	))

	msg0StateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "msg0TID",
			Labels: map[string]string{
				"actor.foo": "inbox",
			},
		},
	}

	require.NoError(t, e.Do(
		flowstate.Commit(
			flowstate.Pause(msg0StateCtx).WithTransit(`inbox`),
		),
	))

	msg1StateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "msg1TID",
			Labels: map[string]string{
				"actor.foo": "inbox",
			},
		},
	}

	require.NoError(t, e.Do(
		flowstate.Commit(
			flowstate.Pause(msg1StateCtx).WithTransit(`inbox`),
		),
	))

	require.Equal(t, []string{
		"actor:actorTID",
		"inbox:msg0TID",
		"inbox:msg1TID",
	}, trkr.WaitVisitedEqual(t, []string{`actor:actorTID`, `inbox:msg0TID`, `inbox:msg1TID`}, time.Millisecond*600))
}
