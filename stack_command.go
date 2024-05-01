package flowstate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Stacked(taskCtx *TaskCtx) bool {
	return taskCtx.Current.Annotations[stackedAnnotation] != ``
}

func Stack(stackedTaskCtx, nextTaskCtx *TaskCtx) *StackCommand {
	return &StackCommand{
		StackedTaskCtx: stackedTaskCtx,
		NextTaskCtx:    nextTaskCtx,
	}

}

var stackedAnnotation = `flowstate.stacked`

type StackCommand struct {
	StackedTaskCtx *TaskCtx
	NextTaskCtx    *TaskCtx
}

func (cmd *StackCommand) Prepare() error {
	b, err := json.Marshal(cmd.StackedTaskCtx)
	if err != nil {
		return fmt.Errorf("json marshal prev task ctx: %s", err)
	}

	stacked := base64.StdEncoding.EncodeToString(b)

	cmd.NextTaskCtx.Current.SetAnnotation(stackedAnnotation, stacked)

	return nil
}
