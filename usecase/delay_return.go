package usecase

import (
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func Delay_Return(t TestingT, d flowstate.Doer, fr flowRegistry) {
	trkr := &Tracker{}

	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		if flowstate.Delayed(stateCtx) {
			return flowstate.Transit(stateCtx, `second`), nil
		}

		return flowstate.Delay(stateCtx, time.Millisecond*200), nil
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
			ID: "aTID",
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))

	time.Sleep(time.Millisecond * 500)

	require.Equal(t, []string{`first`, `first`, `second`}, trkr.Visited())
}
