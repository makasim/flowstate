package usecase

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Fork_WithCommit(t TestingT, d flowstate.Doer, fr flowRegistry) {
	var forkedStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &Tracker{}

	fr.SetFlow("fork", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		forkedStateCtx = &flowstate.StateCtx{}
		stateCtx.CopyTo(forkedStateCtx)
		forkedStateCtx.Current.ID = "forkedTID"
		forkedStateCtx.Current.Rev = 0
		forkedStateCtx.Current.CopyTo(&forkedStateCtx.Committed)

		if err := e.Do(
			flowstate.Commit(
				flowstate.Transit(stateCtx, `origin`),
				flowstate.Transit(forkedStateCtx, `forked`),
			),
			flowstate.Execute(stateCtx),
			flowstate.Execute(forkedStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("forked", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	fr.SetFlow("origin", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `fork`)))
	require.NoError(t, e.Execute(stateCtx))

	time.Sleep(time.Millisecond * 100)

	require.Equal(t, []string{
		`fork`,
		`forked`,
		`origin`,
	}, trkr.VisitedSorted())
}
