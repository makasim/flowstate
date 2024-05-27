package exptcmd

import (
	"fmt"

	"github.com/makasim/flowstate"
)

func Fork(stateCtx, forkedStateCtx *flowstate.StateCtx) *ForkCommand {
	return &ForkCommand{
		StateCtx:       stateCtx,
		ForkedStateCtx: forkedStateCtx,
	}

}

type ForkCommand struct {
	StateCtx       *flowstate.StateCtx
	ForkedStateCtx *flowstate.StateCtx
}

func ForkDoer() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*ForkCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

		tID := cmd.ForkedStateCtx.Current.ID

		cmd.StateCtx.CopyTo(cmd.ForkedStateCtx)

		cmd.ForkedStateCtx.Current.ID = tID
		cmd.ForkedStateCtx.Current.Rev = 0
		//cmd.StateTaskCtx.Current.CopyTo(&forkedTaskCtx.Committed)

		if cmd.ForkedStateCtx.Current.ID == `` {
			return fmt.Errorf(`forked state ID empty`)
		}

		return nil
	})
}
