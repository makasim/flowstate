package tests

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func track(t *testing.T, taskCtx *flowstate.TaskCtx) {
	var visited []interface{}
	visited0, found := taskCtx.Data.Get("visited")
	if found {
		visited = visited0.([]interface{})
	}
	visited = append(visited, string(taskCtx.Transition.ID))

	err := taskCtx.Data.Set("visited", visited)
	require.NoError(t, err)
}

type nopDriver struct {
	calls int
}

func (d *nopDriver) Commit(cmds ...flowstate.Command) error {
	d.calls++
	return nil
}
