package memdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Commiter{}

type Commiter struct {
	l *Log
	e *flowstate.Engine
}

func NewCommiter(l *Log) *Commiter {
	return &Commiter{
		l: l,
	}
}

func (d *Commiter) Do(cmd0 flowstate.Command) error {
	if _, ok := cmd0.(*flowstate.CommitStateCtxCommand); ok {
		return nil
	}

	cmd, ok := cmd0.(*flowstate.CommitCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if len(cmd.Commands) == 0 {
		return fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*flowstate.CommitCommand); ok {
			return fmt.Errorf("commit command not allowed inside another commit")
		}
		if _, ok := c.(*flowstate.ExecuteCommand); ok {
			return fmt.Errorf("execute command not allowed inside commit")
		}
	}

	d.l.Lock()
	defer d.l.Unlock()
	defer d.l.Rollback()

	for _, cmd0 := range cmd.Commands {
		if err := d.e.Do(cmd0); err != nil {
			return fmt.Errorf("%T: do: %w", cmd0, err)
		}

		cmd1, ok := cmd0.(flowstate.CommittableCommand)
		if !ok {
			continue
		}

		stateCtx := cmd1.CommittableStateCtx()
		if stateCtx.Current.ID == `` {
			return fmt.Errorf("state id empty")
		}

		if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
			conflictErr := &flowstate.ErrCommitConflict{}
			conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
			return conflictErr
		}

		d.l.Append(stateCtx)
	}

	d.l.Commit()

	return nil
}

func (d *Commiter) Init(e *flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Commiter) Shutdown(_ context.Context) error {
	return nil
}
