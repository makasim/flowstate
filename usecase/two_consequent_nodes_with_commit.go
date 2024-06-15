package usecase

import (
	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

type initer interface {
	Init(*flowstate.Engine) error
}

type flowRegistry interface {
	SetFlow(id flowstate.FlowID, f flowstate.Flow)
}

func TwoConsequentNodesWithCommit(t TestingT, d flowstate.Doer, fr flowRegistry) {
	trkr := &Tracker{}

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.Transit(stateCtx, `second`),
		), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)

	if d1, ok := d.(initer); ok {
		require.NoError(t, d1.Init(e))
	}

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aTID",
			Rev: 0,
		},
	}

	require.NoError(t, e.Do(flowstate.Commit(
		flowstate.Transit(stateCtx, `first`),
	)))
	require.NoError(t, e.Execute(stateCtx))

	require.Equal(t, []string{`first`, `second`}, trkr.Visited())
}
