package sqlitedriver_test

import (
	"context"
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/stretchr/testify/require"
)

func TestDriver(t *testing.T) {
	d, err := sqlitedriver.New(`:memory:`)
	require.NoError(t, err)

	require.NoError(t, d.Init(&flowstate.Engine{}))

	require.NoError(t, d.Shutdown(context.Background()))
}
