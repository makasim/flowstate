package memdriver

import (
	"context"
	"errors"
	"fmt"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/exptcmd"
	"github.com/makasim/flowstate/stddoer"
)

type Driver struct {
	*FlowRegistry

	l     *Log
	doers []flowstate.Doer
}

func New() *Driver {
	l := &Log{}

	d := &Driver{
		l:            l,
		FlowRegistry: &FlowRegistry{},
	}

	doers := []flowstate.Doer{
		stddoer.Transit(),
		stddoer.Pause(),
		stddoer.Resume(),
		stddoer.End(),
		stddoer.Noop(),

		exptcmd.NewStacker(),
		exptcmd.UnstackDoer(),

		NewFlowGetter(d.FlowRegistry),
		NewCommiter(l),
		NewWatcher(l),
		NewDelayer(),
	}
	d.doers = doers

	return d
}

func (d *Driver) Do(cmd0 flowstate.Command) error {
	if cmd, ok := cmd0.(*flowstate.GetFlowCommand); ok {
		return d.doGetFlow(cmd)
	}

	for _, doer := range d.doers {
		if err := doer.Do(cmd0); errors.Is(err, flowstate.ErrCommandNotSupported) {
			continue
		} else if err != nil {
			return fmt.Errorf("%T: do: %w", doer, err)
		}

		return nil
	}

	return fmt.Errorf("no doer for command %T", cmd0)
}

func (d *Driver) Init(e *flowstate.Engine) error {
	for _, doer := range d.doers {
		if err := doer.Init(e); err != nil {
			return fmt.Errorf("%T: init: %w", doer, err)
		}
	}
	return nil
}

func (d *Driver) Shutdown(_ context.Context) error {
	var res error
	for _, doer := range d.doers {
		if err := doer.Shutdown(context.Background()); err != nil {
			res = errors.Join(res, fmt.Errorf("%T: shutdown: %w", doer, err))
		}
	}

	return res
}

func (d *Driver) doGetFlow(cmd *flowstate.GetFlowCommand) error {
	if cmd.StateCtx.Current.Transition.ToID == "" {
		return fmt.Errorf("transition flow to is empty")
	}

	f, err := d.Flow(cmd.StateCtx.Current.Transition.ToID)
	if err != nil {
		return err
	}

	cmd.Flow = f

	return nil
}
