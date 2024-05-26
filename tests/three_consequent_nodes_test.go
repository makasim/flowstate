package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/stretchr/testify/require"
)

func TestThreeConsequentNodes(t *testing.T) {
	trkr := &tracker2{}

	d := memdriver.New()
	d.SetFlow("first", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `second`), nil
	}))
	d.SetFlow("second", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.Transit(stateCtx, `third`), nil
	}))
	d.SetFlow("third", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx) (flowstate.Command, error) {
		track2(stateCtx, trkr)
		return flowstate.End(stateCtx), nil
	}))

	e := flowstate.NewEngine(d)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "aTID",
			Rev: 0,
		},
	}

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `first`)))
	require.NoError(t, e.Execute(stateCtx))

	require.Equal(t, []string{`first`, `second`, `third`}, trkr.Visited())
}
