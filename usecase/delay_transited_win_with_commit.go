package usecase

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Delay_TransitedWin_WithCommit(t TestingT, d flowstate.Doer, fr flowRegistry) {
	trkr := &Tracker{}

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if flowstate.Delayed(stateCtx) {
			return flowstate.Commit(
				flowstate.Transit(stateCtx, `second`),
			), nil
		}

		if err := e.Do(
			flowstate.Delay(stateCtx, time.Millisecond*200),
		); err != nil {
			return nil, err
		}

		return flowstate.Commit(
			flowstate.Transit(stateCtx, `third`),
		), nil
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
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))

	time.Sleep(time.Millisecond * 500)

	// no second in list
	require.Equal(t, []string{`first`, `third`, `first`}, trkr.Visited())
}
