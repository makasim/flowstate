package usecase

import (
	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Condition(t TestingT, d flowstate.Doer, fr flowRegistry) {
	trkr := &Tracker{}

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		bID := flowstate.FlowID(`third`)
		if stateCtx.Current.Annotations["condition"] == "true" {
			bID = `second`
		}

		return flowstate.Transit(stateCtx, bID), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))
	fr.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)

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
