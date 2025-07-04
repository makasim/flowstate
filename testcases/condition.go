package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Condition(t TestingT, d flowstate.Driver, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{}

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		bID := flowstate.FlowID(`third`)
		if stateCtx.Current.Annotations["condition"] == "true" {
			bID = `second`
		}

		return flowstate.Transit(stateCtx, bID), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))
	fr.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	l, _ := NewTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	// condition true
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTrueTID",
			Annotations: map[string]string{
				"condition": "true",
			},
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))
	require.Equal(t, []string{`first`, `second`}, trkr.Visited())

	// condition false
	trkr.visited = nil

	stateCtx1 := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aFalseTID",
			Annotations: map[string]string{
				"condition": "false",
			},
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx1, `first`)))
	require.NoError(t, e.Execute(stateCtx1))
	require.Equal(t, []string{`first`, `third`}, trkr.Visited())
}
