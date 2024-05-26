package flowstate

import (
	"fmt"
)

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

	return nil
}

type Forker struct {
}

func NewForker() *Forker {
	return &Forker{}
}

func (d *Forker) Do(cmd0 Command) (*StateCtx, error) {
	cmd, ok := cmd0.(*ForkCommand)
	if !ok {
		return nil, ErrCommandNotSupported
	}

	tID := cmd.ForkedStateCtx.Current.ID

	cmd.StateCtx.CopyTo(cmd.ForkedStateCtx)

	cmd.ForkedStateCtx.Current.ID = tID
	cmd.ForkedStateCtx.Current.Rev = 0
	//cmd.StateTaskCtx.Current.CopyTo(&forkedTaskCtx.Committed)

	if cmd.ForkedStateCtx.Current.ID == `` {
		return nil, fmt.Errorf(`forked state ID empty`)
	}

	return nil, nil
}
