package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestDefer_TransitWin_WithCommit(t *testing.T) {
	trkr := &tracker2{}

	d := memdriver.New()
	d.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)

		if flowstate.Deferred(stateCtx) {
			return flowstate.Commit(
				flowstate.Transit(stateCtx, `second`),
			), nil
		}

		if err := e.Do(
			flowstate.Defer(stateCtx, time.Millisecond*200),
		); err != nil {
			return nil, err
		}

		return flowstate.Commit(
			flowstate.Transit(stateCtx, `third`),
		), nil
	}))
	d.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))
	d.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	e := flowstate.NewEngine(d)

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
