package memdriver

import (
	"fmt"

	"github.com/makasim/flowstate"
)

type Driver struct {
	l  Log
	ws []*Watcher
}

func (d *Driver) Do(cmds ...flowstate.Command) error {
	d.l.Lock()
	defer d.l.Unlock()
	defer d.l.Rollback()

	for _, cmd0 := range cmds {
		if err := cmd0.Prepare(); err != nil {
			return fmt.Errorf("%T: prepare: %w", cmd0, err)
		}

		switch cmd := cmd0.(type) {
		case *flowstate.CommitCommand:
			return fmt.Errorf("commit command not allowed")
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
		case *flowstate.WatchCommand:
			w := &Watcher{
				sinceRev:    cmd.SinceRev,
				sinceLatest: cmd.SinceLatest,
				// todo: copy labels
				labels: cmd.Labels,

				watchCh:  make(chan *flowstate.StateCtx, 1),
				changeCh: make(chan int64, 1),
				closeCh:  make(chan struct{}),
				l:        &d.l,
			}

			cmd.Watcher = w
			w.Change(cmd.SinceRev)
			d.ws = append(d.ws, w)

			go w.listen()
		case *flowstate.NopCommand, *flowstate.StackCommand, *flowstate.UnstackCommand, *flowstate.ForkCommand:
			continue
		case *flowstate.ExecuteCommand:
			return fmt.Errorf("execute command not allowed inside commit")
		default:
			return fmt.Errorf("unknown command: %T", cmd0)
		}
	}

	d.l.Commit()

	for _, w := range d.ws {
		w.Change(d.l.rev)
	}

	return nil
}
