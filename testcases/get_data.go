package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func GetData(t *testing.T, e flowstate.Engine, _ flowstate.FlowRegistry, _ flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	stateCtx.SetData(`anAlias`, &flowstate.Data{
		Blob: []byte(`aContent`),
	})
	// guard
	require.NoError(t, e.Do(flowstate.StoreData(stateCtx, `anAlias`)))
	require.NotEmpty(t, stateCtx.Current.Annotations[`flowstate.data.anAlias`])

	expRev := stateCtx.MustData(`anAlias`).Rev

	// load data if not loaded
	stateCtx.Datas = nil
	require.NoError(t, e.Do(flowstate.GetData(stateCtx, `anAlias`)))
	require.Equal(t, expRev, stateCtx.MustData(`anAlias`).Rev)
	require.Equal(t, `aContent`, string(stateCtx.MustData(`anAlias`).Blob))

	// load data if rev not matching
	stateCtx.MustData(`anAlias`).Rev++
	stateCtx.MustData(`anAlias`).Blob = nil
	require.NoError(t, e.Do(flowstate.GetData(stateCtx, `anAlias`)))
	require.Equal(t, expRev, stateCtx.MustData(`anAlias`).Rev)
	require.Equal(t, `aContent`, string(stateCtx.MustData(`anAlias`).Blob))

	// do not load if rev the same
	stateCtx.MustData(`anAlias`).Blob = nil
	require.NoError(t, e.Do(flowstate.GetData(stateCtx, `anAlias`)))
	require.Equal(t, expRev, stateCtx.MustData(`anAlias`).Rev)
	require.Equal(t, ``, string(stateCtx.MustData(`anAlias`).Blob))
}
