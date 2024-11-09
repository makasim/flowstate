package memdriver

import (
	"context"
	"errors"
	"fmt"

	"github.com/makasim/flowstate"
)

type Driver struct {
	*FlowRegistry

	l         *Log
	doers     []flowstate.Doer
	recoverer flowstate.Doer
}

func New(opts ...Option) *Driver {
	l := &Log{}

	d := &Driver{
		l:            l,
		FlowRegistry: &FlowRegistry{},
	}

	doers := []flowstate.Doer{
		flowstate.DefaultTransitDoer,
		flowstate.DefaultPauseDoer,
		flowstate.DefaultResumeDoer,
		flowstate.DefaultEndDoer,
		flowstate.DefaultNoopDoer,
		flowstate.DefaultSerializerDoer,
		flowstate.DefaultDeserializeDoer,
		flowstate.DefaultDereferenceDataDoer,
		flowstate.DefaultReferenceDataDoer,

		NewDataLog(),
		NewFlowGetter(d.FlowRegistry),
		NewCommiter(l),
		NewGetter(l),
		NewWatcher(l),
		NewDelayer(),
	}
	d.doers = doers

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *Driver) Do(cmd0 flowstate.Command) error {
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

	if d.recoverer != nil {
		if err := d.recoverer.Init(e); err != nil {
			return fmt.Errorf("%T: init: %w", d.recoverer, err)
		}
	}

	return nil
}

func (d *Driver) Shutdown(ctx context.Context) error {
	var res error
	for _, doer := range d.doers {
		if err := doer.Shutdown(ctx); err != nil {
			res = errors.Join(res, fmt.Errorf("%T: shutdown: %w", doer, err))
		}
	}

	if d.recoverer != nil {
		if err := d.recoverer.Shutdown(ctx); err != nil {
			res = errors.Join(res, fmt.Errorf("%T: shutdown: %w", d.recoverer, err))
		}
	}

	return res
}

type Option func(*Driver)

func WithRecoverer(r flowstate.Doer) Option {
	return func(d *Driver) {
		d.recoverer = r
	}
}
