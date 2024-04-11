package exptcmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/makasim/flowstate"
)

func Unstack(stateCtx, unstackStateCtx *flowstate.StateCtx) *UnstackCommand {
	return &UnstackCommand{
		StateCtx:        stateCtx,
		UnstackStateCtx: unstackStateCtx,
	}

}

type UnstackCommand struct {
	StateCtx        *flowstate.StateCtx
	UnstackStateCtx *flowstate.StateCtx
}

type Unstacker struct {
}

func UnstackDoer() *Unstacker {
	return &Unstacker{}
}

func (d *Unstacker) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*UnstackCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

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

func (d *Unstacker) Init(_ *flowstate.Engine) error {
	return nil
}

func (d *Unstacker) Shutdown(_ context.Context) error {
	return nil
}
