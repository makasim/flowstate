package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func StoreData(t *testing.T, e *flowstate.Engine, _ flowstate.FlowRegistry, _ flowstate.Driver) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
		},
	}

	// no data
	stateCtx.SetData(`bEmpty`, &flowstate.Data{})
	require.NoError(t, e.Do(flowstate.StoreData(stateCtx, `bEmpty`)))
	require.Greater(t, stateCtx.MustData(`bEmpty`).Rev, int64(0))
	require.NotEmpty(t, stateCtx.MustData(`bEmpty`).Annotations[`checksum/xxhash64`])
	require.NotEmpty(t, stateCtx.Current.Annotations[`flowstate.data.bEmpty`])

	// with data
	stateCtx.SetData(`bNotEmpty`, &flowstate.Data{
		Blob: []byte(`aContent`),
	})
	require.NoError(t, e.Do(flowstate.StoreData(stateCtx, `bNotEmpty`)))
	require.Greater(t, stateCtx.MustData(`bNotEmpty`).Rev, int64(0))
	require.NotEmpty(t, stateCtx.MustData(`bNotEmpty`).Annotations[`checksum/xxhash64`])
	require.NotEmpty(t, stateCtx.Current.Annotations[`flowstate.data.bNotEmpty`])

	// update data without change should not trigger store
	expRev := stateCtx.MustData(`bNotEmpty`).Rev
	require.NoError(t, e.Do(flowstate.StoreData(stateCtx, `bNotEmpty`)))
	require.Equal(t, expRev, stateCtx.MustData(`bNotEmpty`).Rev)

	// update data if it has changed
	expRev = stateCtx.MustData(`bNotEmpty`).Rev
	stateCtx.MustData(`bNotEmpty`).Blob = []byte(`aContentUpdated`)
	require.NoError(t, e.Do(flowstate.StoreData(stateCtx, `bNotEmpty`)))
	require.Greater(t, stateCtx.MustData(`bNotEmpty`).Rev, expRev)
}
