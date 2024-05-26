package tests

import (
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestDefer_Return(t *testing.T) {
	trkr := &tracker2{}

	d := memdriver.New()
	d.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		if flowstate.Deferred(stateCtx) {
			return flowstate.Transit(stateCtx, `second`), nil
		}

		return flowstate.Defer(stateCtx, time.Millisecond*200), nil
	}))
	d.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
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

	require.Equal(t, []string{`first`, `first`, `second`}, trkr.Visited())
}

func TestDefer_EngineDo(t *testing.T) {
	trkr := &tracker2{}

	d := memdriver.New()
	d.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		if flowstate.Deferred(stateCtx) {
			return flowstate.Transit(stateCtx, `second`), nil
		}

		if err := stateCtx.Doer.Do(
			flowstate.Defer(stateCtx, time.Millisecond*200),
		); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	d.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
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

	require.Equal(t, []string{`first`, `first`, `second`}, trkr.Visited())
}
