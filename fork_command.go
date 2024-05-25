package flowstate

import "fmt"

func Fork(stateCtx, forkedStateCtx *StateCtx) *ForkCommand {
	return &ForkCommand{
		StateCtx:       stateCtx,
		ForkedStateCtx: forkedStateCtx,
	}

}

type ForkCommand struct {
	StateCtx       *StateCtx
	ForkedStateCtx *StateCtx
}

func (cmd *ForkCommand) Prepare() error {
	tID := cmd.ForkedStateCtx.Current.ID

	cmd.StateCtx.CopyTo(cmd.ForkedStateCtx)

	cmd.ForkedStateCtx.Current.ID = tID
	cmd.ForkedStateCtx.Current.Rev = 0
	//cmd.StateTaskCtx.Current.CopyTo(&forkedTaskCtx.Committed)

	if cmd.ForkedStateCtx.Current.ID == `` {
		return fmt.Errorf(`forked state ID empty`)
	}

	return nil
}
