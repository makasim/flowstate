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

	return nil
}

type Unstacker struct {
}

func NewUnstacker() *Unstacker {
	return &Unstacker{}
}

func (d *Unstacker) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*UnstackCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	stackedTask := cmd.StateCtx.Current.Annotations[stackedAnnotation]
	if stackedTask == `` {
		return nil, fmt.Errorf("no state to unstack; the annotation not set")
	}

	b, err := base64.StdEncoding.DecodeString(stackedTask)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %s", err)
	}

	if err := json.Unmarshal(b, cmd.UnstackStateCtx); err != nil {
		return nil, fmt.Errorf("json unmarshal: %s", err)
	}

	cmd.StateCtx.Current.Annotations[stackedAnnotation] = ``

	return nil, nil
}
