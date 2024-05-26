package memdriver

import (
	"fmt"

	"github.com/makasim/flowstate"
)

type Commiter struct {
	l    *Log
	doer flowstate.Doer
}

func NewCommiter(l *Log, doer flowstate.Doer) *Commiter {
	return &Commiter{
		l:    l,
		doer: doer,
	}
}

func (d *Commiter) Do(cmd0 flowstate.Command) (*flowstate.StateCtx, error) {
	cmd, ok := cmd0.(*flowstate.CommitCommand)
	if !ok {
		return nil, flowstate.ErrCommandNotSupported
	}

	if len(cmd.Commands) == 0 {
		return nil, fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*flowstate.CommitCommand); ok {
			return nil, fmt.Errorf("commit command in commit command not allowed")
		}
	}

	d.l.Lock()
	defer d.l.Unlock()
	defer d.l.Rollback()

	for _, cmd0 := range cmd.Commands {
		if stateCtx, err := d.doer.Do(cmd0); err != nil {
			return nil, fmt.Errorf("%T: do: %w", cmd0, err)
		} else if stateCtx != nil {
			cmd.NextStateCtxs = append(cmd.NextStateCtxs, stateCtx)
		}

		switch cmd := cmd0.(type) {
		case *flowstate.CommitCommand:
			return nil, fmt.Errorf("commit command not allowed")
		case *flowstate.TransitCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return nil, fmt.Errorf("state id empty")
			}

			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return nil, conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.EndCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return nil, fmt.Errorf("state id empty")
			}
			if stateCtx.Committed.Rev == 0 {
				return nil, fmt.Errorf("state rev empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return nil, conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.DeferCommand:
			stateCtx := cmd.DeferredStateCtx

			if stateCtx.Current.ID == `` {
				return nil, fmt.Errorf("state id empty")
			}
			if stateCtx.Committed.Rev == 0 {
				return nil, fmt.Errorf("state rev empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return nil, conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.PauseCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return nil, fmt.Errorf("state id empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return nil, conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.ResumeCommand:
			stateCtx := cmd.StateCtx

			if stateCtx.Current.ID == `` {
				return nil, fmt.Errorf("state id empty")
			}
			if _, rev := d.l.LatestByID(stateCtx.Current.ID); rev != stateCtx.Committed.Rev {
				conflictErr := &flowstate.ErrCommitConflict{}
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				return nil, conflictErr
			}

			d.l.Append(stateCtx)
		case *flowstate.NoopCommand, *flowstate.StackCommand, *flowstate.UnstackCommand, *flowstate.ForkCommand:
			continue
		case *flowstate.ExecuteCommand:
			continue
		case *flowstate.GetFlowCommand:
			continue
		default:
			return nil, fmt.Errorf("unknown command: %T", cmd0)
		}
	}

	d.l.Commit()

	return nil, nil
}
