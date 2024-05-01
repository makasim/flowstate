package flowstate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Unstack(taskCtx, unstackTaskCtx *TaskCtx) *UnstackCommand {
	return &UnstackCommand{
		TaskCtx:        taskCtx,
		UnstackTaskCtx: unstackTaskCtx,
	}

}

type UnstackCommand struct {
	TaskCtx        *TaskCtx
	UnstackTaskCtx *TaskCtx
}

func (cmd *UnstackCommand) Prepare() error {
	stackedTask := cmd.TaskCtx.Current.Annotations[stackedAnnotation]
	if stackedTask == `` {
		return fmt.Errorf("no task to unstack; the annotation not set")
	}

	b, err := base64.StdEncoding.DecodeString(stackedTask)
	if err != nil {
		return fmt.Errorf("base64 decode: %s", err)
	}

	if err := json.Unmarshal(b, cmd.UnstackTaskCtx); err != nil {
		return fmt.Errorf("json unmarshal: %s", err)
	}

	cmd.TaskCtx.Current.Annotations[stackedAnnotation] = ``

	return nil
}
