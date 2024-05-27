package memdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Commiter{}

type Commiter struct {
	l *Log
	e flowstate.Engine
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

		switch cmd := cmd0.(type) {
		case *flowstate.TransitCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return fmt.Errorf("state id empty")
			}

			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.EndCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return fmt.Errorf("state id empty")
			}
			if stateCtx.Committed.Rev == 0 {
				return fmt.Errorf("state rev empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.DeferCommand:
			stateCtx := cmd.DeferredStateCtx

			if stateCtx.Current.ID == `` {
				return fmt.Errorf("state id empty")
			}
			if stateCtx.Committed.Rev == 0 {
				return fmt.Errorf("state rev empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.PauseCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return fmt.Errorf("state id empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.ResumeCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return fmt.Errorf("state id empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.NoopCommand, *flowstate.StackCommand, *flowstate.UnstackCommand, *flowstate.ForkCommand:
			continue
		case *flowstate.GetFlowCommand:
			continue
		default:
			return fmt.Errorf("unknown command: %T", cmd0)
		}
	}

	d.l.Commit()

	return nil
}

func (d *Commiter) Init(e flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Commiter) Shutdown(ctx context.Context) error {
	return nil
}
