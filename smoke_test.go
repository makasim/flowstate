package flowstate_test

import (
	"testing"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func TestSimple(t *testing.T) {
	p := flowstate.Process{
		ID:  "simplePID",
		Rev: 1,
		Nodes: []flowstate.Node{
			{
				ID:         "plusNID",
				BehaviorID: "plus",
			},
		},
		Transitions: []flowstate.Transition{
			{
				ID:     "plusTID",
				FromID: "",
				ToID:   "plusNID",
			},
		},
	}

	br := &flowstate.MapBehaviorRegistry{}
	br.SetBehavior("plus", flowstate.BehaviorFunc(func(taskCtx *flowstate.TaskCtx) error {
		a0, _ := taskCtx.Data.Get("a")
		a := a0.(float64)
		b0, _ := taskCtx.Data.Get("b")
		b := b0.(float64)

		if err := taskCtx.Data.Set("result", a+b); err != nil {
			return err
		}

		return nil // todo
	}))

	e := flowstate.NewEngine(br)

	task := flowstate.Task{
		ID:         "simpleTID",
		Rev:        0,
		ProcessID:  p.ID,
		ProcessRev: p.Rev,

		Transition: p.Transitions[0],
	}

	taskCtx := &flowstate.TaskCtx{
		Task:    task,
		Process: p,
		Node:    p.Nodes[0],
		Data: flowstate.Data{
			Bytes: []byte(`{"a": 123, "b": 456}`),
		},
	}

	require.NoError(t, e.Execute(taskCtx))

	result, _ := taskCtx.Data.Get("result")
	require.Equal(t, float64(579), result)

}
