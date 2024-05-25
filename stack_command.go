package flowstate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Stacked(stateCtx *StateCtx) bool {
	return stateCtx.Current.Annotations[stackedAnnotation] != ``
}

func Stack(stackedStateCtx, nextStateCtx *StateCtx) *StackCommand {
	return &StackCommand{
		StackedStateCtx: stackedStateCtx,
		NextStateCtx:    nextStateCtx,
	}

}

var stackedAnnotation = `flowstate.stacked`

type StackCommand struct {
	StackedStateCtx *StateCtx
	NextStateCtx    *StateCtx
}

func (cmd *StackCommand) Prepare() error {
	if Stacked(cmd.NextStateCtx) {
		return fmt.Errorf("next state already stacked")
	}

	b, err := json.Marshal(cmd.StackedStateCtx)
	if err != nil {
		return fmt.Errorf("json marshal prev state ctx: %s", err)
	}

	stacked := base64.StdEncoding.EncodeToString(b)

	cmd.NextStateCtx.Current.SetAnnotation(stackedAnnotation, stacked)

	return nil
}
