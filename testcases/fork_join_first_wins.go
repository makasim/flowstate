package testcases

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func ForkJoin_FirstWins(t TestingT, d flowstate.Doer, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	var forkedStateCtx *flowstate.StateCtx
	var forkedTwoStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &Tracker{}

	fr.SetFlow("fork", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		stateCtx.Current.SetLabel(`theForkJoinLabel`, string(stateCtx.Current.ID))
		forkedStateCtx = stateCtx.NewTo(`forkedTID`, &flowstate.StateCtx{})
		forkedTwoStateCtx = stateCtx.NewTo(`forkedTwoTID`, &flowstate.StateCtx{})

		if err := e.Do(
			flowstate.Commit(
				flowstate.Transit(stateCtx, `forked`),
				flowstate.Transit(forkedStateCtx, `forked`),
				flowstate.Transit(forkedTwoStateCtx, `forked`),
			),
			flowstate.Execute(stateCtx),
			flowstate.Execute(forkedStateCtx),
			flowstate.Execute(forkedTwoStateCtx),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("join", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if stateCtx.Committed.Transition.ToID != `join` {
			if err := e.Do(flowstate.Commit(
				flowstate.Transit(stateCtx, `join`),
			)); err != nil {
				return nil, err
			}
		}

		lis, err := flowstate.DoWatch(e, flowstate.Watch(map[string]string{
			`theForkJoinLabel`: stateCtx.Current.Labels[`theForkJoinLabel`],
		}))
		if err != nil {
			return nil, err
		}
		defer lis.Close()

		cnt := 0
		for {
			select {
			case <-stateCtx.Done():
				return flowstate.Noop(stateCtx), nil
			case changedState := <-lis.Listen():
				changedStateCtx := changedState.CopyToCtx(&flowstate.StateCtx{})

				if changedStateCtx.Current.Transition.ToID != `join` {
					continue
				}
				cnt++

				if changedStateCtx.Current.ID != stateCtx.Current.ID {
					continue
				}

				if cnt == 1 {
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

	fr.SetFlow("forked", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `join`), nil
	}))

	fr.SetFlow("joined", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
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
