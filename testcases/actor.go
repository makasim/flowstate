package testcases

import (
	"fmt"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Actor(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	trkr := &Tracker{IncludeTaskID: true}

	mustSetFlow(fr, "actor", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
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
	mustSetFlow(fr, "inbox", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		return nil, fmt.Errorf("must never be executed")
	}))

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
