package memdriver

import (
	"errors"
	"fmt"

	"github.com/makasim/flowstate"
)

type Driver struct {
	*FlowRegistry
	*Log
	doers []flowstate.Doer
}

func New() *Driver {
	fr := NewFlowRegistry()
	l := &Log{}

	d := &Driver{
		Log:          l,
		FlowRegistry: fr,
	}

	doers := []flowstate.Doer{
		flowstate.NewTransiter(),
		flowstate.NewPauser(),
		flowstate.NewResumer(),
		flowstate.NewEnder(),
		flowstate.NewNooper(),
		flowstate.NewForker(),
		flowstate.NewStacker(),
		flowstate.NewUnstacker(),
		flowstate.NewExecutor(d),

		fr,
		NewCommiter(l, d),
		NewWatcher(l),
		NewDeferer(d),
	}
	d.doers = doers

	return d
}

func (d *Driver) Do(cmd0 flowstate.Command) (*flowstate.StateCtx, error) {
	for _, doer := range d.doers {
		nextStateCtx, err := doer.Do(cmd0)
		if errors.Is(err, flowstate.ErrCommandNotSupported) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("%T: do: %w", doer, err)
		}

		return nextStateCtx, nil
	}

	return nil, fmt.Errorf("no doer for command %T", cmd0)
}
