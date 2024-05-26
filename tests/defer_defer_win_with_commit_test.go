package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestDefer_DeferWin_WithCommit(t *testing.T) {
	trkr := &tracker2{}

	fr := memdriver.NewFlowRegistry()
	fr.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
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

		time.Sleep(time.Millisecond * 300)

		return flowstate.Commit(
			flowstate.Transit(stateCtx, `third`),
		), nil
	}))
	fr.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))
	fr.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	d := &memdriver.Driver{}
	e := flowstate.NewEngine(d, fr)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}
	stateCtx.Current.CopyTo(&stateCtx.Committed)

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))

	time.Sleep(time.Millisecond * 500)

	// no third in list
	require.Equal(t, []string{`first`, `first`, `second`}, trkr.Visited())
}
