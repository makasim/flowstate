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

		var commitStateCtx *flowstate.StateCtx
		switch cmd := cmd0.(type) {
		case *flowstate.TransitCommand:
			commitStateCtx = cmd.StateCtx
		case *flowstate.EndCommand:
			commitStateCtx = cmd.StateCtx
		case *flowstate.DelayCommand:
			commitStateCtx = cmd.DelayStateCtx
		case *flowstate.PauseCommand:
			commitStateCtx = cmd.StateCtx
		case *flowstate.ResumeCommand:
			commitStateCtx = cmd.StateCtx
		default:
			continue
		}

		if commitStateCtx != nil {
			if commitStateCtx.Current.ID == `` {
				return fmt.Errorf("state id empty")
			}

			if _, rev := d.l.LatestByID(commitStateCtx.Current.ID); rev != commitStateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), commitStateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return conflictErr
			}

			d.l.Append(commitStateCtx)
		}
	}

	d.l.Commit()

	return nil
}

func (d *Commiter) Init(e *flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Commiter) Shutdown(ctx context.Context) error {
	return nil
}
