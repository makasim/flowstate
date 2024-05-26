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

type Stacker struct {
}

func NewStacker() *Stacker {
	return &Stacker{}
}

func (d *Stacker) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*StackCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	if Stacked(cmd.NextStateCtx) {
		return nil, fmt.Errorf("next state already stacked")
	}

	b, err := json.Marshal(cmd.StackedStateCtx)
	if err != nil {
		return nil, fmt.Errorf("json marshal prev state ctx: %s", err)
	}

	stacked := base64.StdEncoding.EncodeToString(b)

	cmd.NextStateCtx.Current.SetAnnotation(stackedAnnotation, stacked)

	return nil, nil
}
