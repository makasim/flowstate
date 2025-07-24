package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func DataStoreGet(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &Tracker{}

	expData := &flowstate.Data{
		ID:  `aDataID`,
		Rev: 1,
		B:   []byte(`foo`),
	}
	actData := &flowstate.Data{}

	mustSetFlow(fr, "store", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		d := &flowstate.Data{
			ID: `aDataID`,
			B:  []byte(`foo`),
		}

		if err := e.Do(
			flowstate.AttachData(stateCtx, d, `aDataKey`),
			flowstate.Transit(stateCtx, `get`),
		); err != nil {
			return nil, err
		}

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(fr, "get", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(
			flowstate.GetData(stateCtx, actData, `aDataKey`),
			flowstate.Transit(stateCtx, `end`),
		); err != nil {
			return nil, err
		}

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(fr, "end", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Park(stateCtx), nil
	}))

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `store`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitVisitedEqual(t, []string{`store`, `get`, `end`}, time.Second*2)
	require.Equal(t, expData, actData)
}
