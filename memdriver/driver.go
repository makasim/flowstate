package memdriver

import (
	"fmt"
	"sync"

	"github.com/makasim/flowstate"
)

type Driver struct {
	mux sync.Mutex
	rev int64
	s   map[flowstate.TaskID]flowstate.Task
}

func (d *Driver) Do(cmds ...flowstate.Command) error {
	d.mux.Lock()
	defer d.mux.Unlock()

	changes := make(map[flowstate.TaskID]flowstate.Task)

	for _, cmd0 := range cmds {
		if err := cmd0.Prepare(); err != nil {
			return fmt.Errorf("%T: prepare: %w", cmd0, err)
		}

		switch cmd := cmd0.(type) {
		case *flowstate.CommitCommand:
			return fmt.Errorf("commit command not allowed")
		case *flowstate.TransitCommand:
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if d.s[taskCtx.Committed.ID].Rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.commit(changes, taskCtx)
		case *flowstate.EndCommand:
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if taskCtx.Committed.Rev == 0 {
				return fmt.Errorf("task rev empty")
			}
			if d.s[taskCtx.Committed.ID].Rev != taskCtx.Current.Rev {
				return flowstate.ErrCommitConflict
			}

			d.commit(changes, taskCtx)
		case *flowstate.DeferCommand:
			taskCtx := cmd.DeferredTaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if taskCtx.Committed.Rev == 0 {
				return fmt.Errorf("task rev empty")
			}
			if d.s[taskCtx.Committed.ID].Rev != taskCtx.Current.Rev {
				return flowstate.ErrCommitConflict
			}

			d.commit(changes, taskCtx)
		case *flowstate.PauseCommand:
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if d.s[taskCtx.Committed.ID].Rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.commit(changes, taskCtx)
		case *flowstate.ResumeCommand:
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if d.s[taskCtx.Committed.ID].Rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.commit(changes, taskCtx)
		case *flowstate.WatchCommand:
			w :=
		case *flowstate.NopCommand, *flowstate.StackCommand, *flowstate.UnstackCommand:
			continue
		default:
			return fmt.Errorf("unknown command: %T", cmd0)
		}
	}

	if d.s == nil {
		d.s = make(map[flowstate.TaskID]flowstate.Task)
	}

	for _, t := range changes {
		d.s[t.ID] = t
	}

	return nil
}

func (d *Driver) commit(s map[flowstate.TaskID]flowstate.Task, taskCtx *flowstate.TaskCtx) {
	committedT := s[taskCtx.Committed.ID]
	taskCtx.Current.CopyTo(&committedT)

	d.rev++
	committedT.Rev = d.rev
	s[committedT.ID] = committedT

	committedT.CopyTo(&taskCtx.Committed)
	committedT.CopyTo(&taskCtx.Current)

	// emulate transitions storing during commit
	taskCtx.Transitions = taskCtx.Transitions[:0]
}
