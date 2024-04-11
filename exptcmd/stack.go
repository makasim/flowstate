package exptcmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/makasim/flowstate"
)

var stackedAnnotation = `flowstate.stacked`

func Stacked(stateCtx *flowstate.StateCtx) bool {
	return stateCtx.Current.Annotations[stackedAnnotation] != ``
}

func Stack(stackedStateCtx, nextStateCtx *flowstate.StateCtx) *StackCommand {
	return &StackCommand{
		StackedStateCtx: stackedStateCtx,
		NextStateCtx:    nextStateCtx,
	}

}

type StackCommand struct {
	StackedStateCtx *flowstate.StateCtx
	NextStateCtx    *flowstate.StateCtx
}

func NewStacker() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*StackCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

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
	})
}
