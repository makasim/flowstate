package flowstate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Unstack(stateCtx, unstackStateCtx *StateCtx) *UnstackCommand {
	return &UnstackCommand{
		StateCtx:        stateCtx,
		UnstackStateCtx: unstackStateCtx,
	}

}

type UnstackCommand struct {
	StateCtx        *StateCtx
	UnstackStateCtx *StateCtx
}

func (cmd *UnstackCommand) Prepare() error {
	stackedTask := cmd.StateCtx.Current.Annotations[stackedAnnotation]
	if stackedTask == `` {
		return fmt.Errorf("no state to unstack; the annotation not set")
	}

	b, err := base64.StdEncoding.DecodeString(stackedTask)
	if err != nil {
		return fmt.Errorf("base64 decode: %s", err)
	}

	if err := json.Unmarshal(b, cmd.UnstackStateCtx); err != nil {
		return fmt.Errorf("json unmarshal: %s", err)
	}

	cmd.StateCtx.Current.Annotations[stackedAnnotation] = ``

	return nil
}
