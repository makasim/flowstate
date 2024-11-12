package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func DataStoreGetWithCommit(t TestingT, d flowstate.Doer, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

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

	fr.SetFlow("store", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
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
	fr.SetFlow("get", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
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
	fr.SetFlow("end", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
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

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `store`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitVisitedEqual(t, []string{`store`, `get`, `end`}, time.Second*2)
	require.Equal(t, expData, actData)
}
