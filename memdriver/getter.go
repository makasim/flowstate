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
	cmd, ok := cmd0.(*flowstate.GetCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

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

func (d *Getter) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Getter) Shutdown(_ context.Context) error {
	return nil
}
