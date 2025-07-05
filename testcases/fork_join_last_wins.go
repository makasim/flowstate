package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func ForkJoin_LastWins(t TestingT, d flowstate.Driver) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &Tracker{}

	mustSetFlow(d, "fork", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		stateCtx.Current.SetLabel(`theForkJoinLabel`, string(stateCtx.Current.ID))

		forkedStateCtx := stateCtx.NewTo(`forkedTID`, &flowstate.StateCtx{})
		forkedTwoStateCtx := stateCtx.NewTo(`forkedTwoTID`, &flowstate.StateCtx{})
		copyStateCtx := stateCtx.CopyTo(&flowstate.StateCtx{})

		if err := e.Do(
			flowstate.Commit(
				flowstate.Transit(copyStateCtx, `forked`),
				flowstate.Transit(forkedStateCtx, `forked`),
				flowstate.Transit(forkedTwoStateCtx, `forked`),
			),
			flowstate.Execute(copyStateCtx),
			flowstate.Execute(forkedStateCtx),
			flowstate.Execute(forkedTwoStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	mustSetFlow(d, "join", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if stateCtx.Committed.Transition.To != `join` {
			if err := e.Do(flowstate.Commit(
				flowstate.Transit(stateCtx, `join`),
			)); err != nil {
				return nil, err
			}
		}

		w := flowstate.NewWatcher(e, flowstate.GetStatesByLabels(map[string]string{
			`theForkJoinLabel`: stateCtx.Current.Labels[`theForkJoinLabel`],
		}))
		defer w.Close()

		cnt := 0
		for {

			select {
			case <-stateCtx.Done():
				return flowstate.Noop(stateCtx), nil
			case changedState := <-w.Next():
				changedStateCtx := changedState.CopyToCtx(&flowstate.StateCtx{})

				if changedStateCtx.Current.Transition.To != `join` {
					continue
				}
				cnt++

				if changedStateCtx.Current.ID != stateCtx.Current.ID {
					continue
				}

				if cnt == 3 {
					return flowstate.Commit(
						flowstate.Transit(stateCtx, `joined`),
					), nil
				}

				return flowstate.Commit(
					flowstate.End(stateCtx),
				), nil
			}
		}
	}))

	mustSetFlow(d, "forked", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `join`), nil
	}))

	mustSetFlow(d, "joined", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
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

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `fork`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitSortedVisitedEqual(t, []string{
		"fork",
		"forked",
		"forked",
		"forked",
		"join",
		"join",
		"join",
		"joined",
	}, time.Second)
}
