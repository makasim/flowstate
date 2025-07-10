package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func DataStoreGetWithCommit(t *testing.T, e flowstate.Engine, d flowstate.Driver) {
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

	mustSetFlow(d, "store", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		d := &flowstate.Data{
			ID: `aDataID`,
			B:  []byte(`foo`),
		}

		if err := e.Do(flowstate.Commit(
			flowstate.StoreData(d),
			flowstate.ReferenceData(stateCtx, d, `aDataKey`),
			flowstate.Transit(stateCtx, `get`),
		)); err != nil {
			return nil, err
		}

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(d, "get", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(flowstate.Commit(
			flowstate.DereferenceData(stateCtx, actData, `aDataKey`),
			flowstate.GetData(actData),
			flowstate.Transit(stateCtx, `end`),
		)); err != nil {
			return nil, err
		}

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(d, "end", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `store`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitVisitedEqual(t, []string{`store`, `get`, `end`}, time.Second*2)
	require.Equal(t, expData, actData)
}
