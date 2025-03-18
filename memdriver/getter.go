package memdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Getter{}

type Getter struct {
	l *Log
}

func NewGetter(l *Log) *Getter {
	return &Getter{
		l: l,
	}
}

func (d *Getter) Do(cmd0 flowstate.Command) error {
	switch cmd := cmd0.(type) {
	case *flowstate.GetCommand:
		return d.doGet(cmd)
	case *flowstate.GetManyCommand:
		return d.doGetMany(cmd)
	default:
		return flowstate.ErrCommandNotSupported
	}
}

func (d *Getter) doGet(cmd *flowstate.GetCommand) error {
	if len(cmd.Labels) > 0 {
		stateCtx, _ := d.l.GetLatestByLabels([]map[string]string{cmd.Labels})
		if stateCtx == nil {
			return fmt.Errorf("state not found by labels %v", cmd.Labels)
		}
		stateCtx.CopyTo(cmd.StateCtx)
	} else if cmd.ID != "" && cmd.Rev == 0 {
		stateCtx, _ := d.l.GetLatestByID(cmd.ID)
		if stateCtx == nil {
			return fmt.Errorf("state not found by id %s", cmd.ID)
		}
		stateCtx.CopyTo(cmd.StateCtx)
	} else if cmd.ID != "" && cmd.Rev > 0 {
		stateCtx := d.l.GetByIDAndRev(cmd.ID, cmd.Rev)
		if stateCtx == nil {
			return fmt.Errorf("state not found by id %s and rev %d", cmd.ID, cmd.Rev)
		}
		stateCtx.CopyTo(cmd.StateCtx)
	} else {
		return fmt.Errorf("invalid get command")
	}

	return nil
}

func (d *Getter) doGetMany(cmd *flowstate.GetManyCommand) error {
	cmd.Prepare()

	states := make([]flowstate.State, 0, cmd.Limit)
	limit := cmd.Limit + 1

	sinceRev := cmd.SinceRev
	if sinceRev == -1 {
		d.l.Lock()
		var stateCtx *flowstate.StateCtx
		stateCtx, sinceRev = d.l.GetLatestByLabels(cmd.Labels)
		if stateCtx != nil {
			states = append(states, stateCtx.Committed)
		}
		d.l.Unlock()
	}

	d.l.Lock()
	untilRev := d.l.rev
	d.l.Unlock()

	for {
		var logStates []*flowstate.StateCtx

		d.l.Lock()
		logStates, sinceRev = d.l.Entries(sinceRev, limit)
		d.l.Unlock()

		if len(logStates) == 0 {
			cmd.SetResult(&flowstate.GetManyResult{
				States: states,
			})
			return nil
		}

		for _, s := range logStates {
			if !cmd.SinceTime.IsZero() && s.Committed.CommittedAtUnixMilli < cmd.SinceTime.UnixMilli() {
				continue
			}
			if !matchLabels(s.Committed, cmd.Labels) {
				continue
			}

			limit--
			states = append(states, s.Committed)
		}

		if len(states) >= limit {
			cmd.SetResult(&flowstate.GetManyResult{
				States: states[:cmd.Limit],
				More:   len(states) > cmd.Limit,
			})
			return nil
		} else if sinceRev >= untilRev {
			cmd.SetResult(&flowstate.GetManyResult{
				States: states,
				More:   false,
			})
			return nil
		}
	}
}

func (d *Getter) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Getter) Shutdown(_ context.Context) error {
	return nil
}
