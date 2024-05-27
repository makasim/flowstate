package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/exptcmd"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestForkJoin_LastWins(t *testing.T) {
	var forkedStateCtx *flowstate.StateCtx
	var forkedTwoStateCtx *flowstate.StateCtx
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	trkr := &tracker2{}

	d := memdriver.New()
	d.SetFlow("fork", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		stateCtx.Current.SetLabel(`theForkJoinLabel`, string(stateCtx.Current.ID))

		forkedStateCtx = &flowstate.StateCtx{}
		forkedStateCtx.Current.ID = "forkedTID"
		forkedStateCtx.Committed.ID = "forkedTID"

		forkedTwoStateCtx = &flowstate.StateCtx{}
		forkedTwoStateCtx.Current.ID = "forkedTwoTID"
		forkedTwoStateCtx.Committed.ID = "forkedTwoTID"

		if err := e.Do(
			flowstate.Commit(
				exptcmd.Fork(stateCtx, forkedStateCtx),
				exptcmd.Fork(stateCtx, forkedTwoStateCtx),

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
	d.SetFlow("join", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		if stateCtx.Committed.Transition.ToID != `join` {
			if err := e.Do(flowstate.Commit(
				flowstate.Transit(stateCtx, `join`),
			)); err != nil {
				return nil, err
			}
		}

		w, err := e.Watch(0, map[string]string{
			`theForkJoinLabel`: stateCtx.Current.Labels[`theForkJoinLabel`],
		})
		if err != nil {
			return nil, err
		}
		defer w.Close()

		cnt := 0
		for {

			select {
			case changedStateCtx := <-w.Watch():
				if changedStateCtx.Current.Transition.ToID != `join` {
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

	d.SetFlow("forked", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `join`), nil
	}))

	d.SetFlow("joined", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `fork`)))
	require.NoError(t, e.Execute(stateCtx))

	time.Sleep(time.Millisecond * 100)

	require.Equal(t, []string{
		"fork",
		"forked",
		"forked",
		"forked",
		"join",
		"join",
		"join",
		"joined",
	}, trkr.VisitedSorted())
}
