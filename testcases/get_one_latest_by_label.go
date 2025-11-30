package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetOneLatestByLabel(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
			Labels: map[string]string{
				"foo": "fooVal",
			},
		},
	}

	stateCtx.Current.SetAnnotation("v", "1")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))
	stateCtx.Current.SetAnnotation("v", "2")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))
	stateCtx.Current.SetAnnotation("v", "3")
	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Park(stateCtx),
	)))
	expStateCtx := stateCtx.CopyTo(&flowstate.StateCtx{})
	expStateCtx.Transitions = nil

	foundStateCtx := &flowstate.StateCtx{}

	require.NoError(t, e.Do(flowstate.GetStateByLabels(foundStateCtx, map[string]string{
		"foo": "fooVal",
	})))

	require.Equal(t, expStateCtx, foundStateCtx)
}
