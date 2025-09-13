package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func DataStoreGetWithCommit(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &Tracker{}

	expData := &flowstate.Data{
		Rev: 1,
		Annotations: map[string]string{
			"checksum/xxhash64": "3728699739546630719",
		},
		Blob: []byte(`foo`),
	}
	actData := &flowstate.Data{}

	mustSetFlow(fr, "store", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		d := &flowstate.Data{
			Blob: []byte(`foo`),
		}
		stateCtx.SetData(`aDataKey`, d)

		if err := e.Do(flowstate.Commit(
			flowstate.StoreData(stateCtx, `aDataKey`),
			flowstate.Transit(stateCtx, `get`),
		)); err != nil {
			return nil, err
		}

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(fr, "get", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(flowstate.Commit(
			flowstate.GetData(stateCtx, `aDataKey`),
			flowstate.Transit(stateCtx, `end`),
		)); err != nil {
			return nil, err
		}
		actData = stateCtx.MustData(`aDataKey`)

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
