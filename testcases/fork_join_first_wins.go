package testcases

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func ForkJoin_FirstWins(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &Tracker{}

	mustSetFlow(fr, "fork", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
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

		return flowstate.Noop(), nil
	}))
	mustSetFlow(fr, "join", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if stateCtx.Committed.Transition.To != `join` {
			if err := e.Do(flowstate.Commit(
				flowstate.Transit(stateCtx, `join`),
			)); err != nil {
				return nil, err
			}
		}

		// time.Millisecond*100
		w := e.Watch(flowstate.GetStatesByLabels(map[string]string{
			`theForkJoinLabel`: stateCtx.Current.Labels[`theForkJoinLabel`],
		}))
		defer w.Close()

		cnt := 0
		for {
			select {
			case <-stateCtx.Done():
				return flowstate.Noop(), nil
			case changedState := <-w.Next():
				changedStateCtx := changedState.CopyToCtx(&flowstate.StateCtx{})

				if changedStateCtx.Current.Transition.To != `join` {
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
					flowstate.Park(stateCtx),
				), nil

			}
		}
	}))

	mustSetFlow(fr, "forked", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `join`), nil
	}))

	mustSetFlow(fr, "joined", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Park(stateCtx), nil
	}))

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
