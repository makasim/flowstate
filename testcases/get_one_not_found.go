package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetOneNotFound(t *testing.T, e *flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	err := e.Do(flowstate.GetStateByID(&flowstate.StateCtx{}, `notExist`, 0))
	require.ErrorIs(t, err, flowstate.ErrNotFound)
}
