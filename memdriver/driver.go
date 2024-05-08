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
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}

			if _, rev := d.l.Latest(taskCtx.Committed.ID); rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.l.Append(taskCtx)
		case *flowstate.EndCommand:
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if taskCtx.Committed.Rev == 0 {
				return fmt.Errorf("task rev empty")
			}
			if _, rev := d.l.Latest(taskCtx.Committed.ID); rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.l.Append(taskCtx)
		case *flowstate.DeferCommand:
			taskCtx := cmd.DeferredTaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if taskCtx.Committed.Rev == 0 {
				return fmt.Errorf("task rev empty")
			}
			if _, rev := d.l.Latest(taskCtx.Committed.ID); rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.l.Append(taskCtx)
		case *flowstate.PauseCommand:
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if _, rev := d.l.Latest(taskCtx.Committed.ID); rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.l.Append(taskCtx)
		case *flowstate.ResumeCommand:
			taskCtx := cmd.TaskCtx

			if taskCtx.Committed.ID == `` {
				return fmt.Errorf("task id empty")
			}
			if _, rev := d.l.Latest(taskCtx.Committed.ID); rev != taskCtx.Committed.Rev {
				return flowstate.ErrCommitConflict
			}

			d.l.Append(taskCtx)
		case *flowstate.WatchCommand:
			w := &Watcher{
				watchCh:  make(chan *flowstate.TaskCtx, 1),
				changeCh: make(chan int64, 1),
				closeCh:  make(chan struct{}),
			}

			since := cmd.Since
			// todo: copy labels
			labels := cmd.Labels

			d.ws = append(d.ws, w)

			go func() {
				var tasks []*flowstate.TaskCtx

			skip:
				for {
					select {
					case <-w.changeCh:
						d.l.Lock()
						tasks, since = d.l.Entries(since, 10)
						d.l.Unlock()

						if len(tasks) == 0 {
							continue skip
						}

					next:
						for _, t := range tasks {
							for k, v := range labels {
								if t.Committed.Labels[k] != v {
									continue next
								}
							}

							select {
							case w.watchCh <- t:
								continue next
							case <-w.closeCh:
								return
							}
						}
					case <-w.closeCh:
						return
					}
				}
			}()

			cmd.Watcher = w
		case *flowstate.NopCommand, *flowstate.StackCommand, *flowstate.UnstackCommand:
			continue
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
