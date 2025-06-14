package badgerdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

type Driver struct {
	db  *badger.DB
	seq *badger.Sequence

	doers []flowstate.Doer

	l *slog.Logger
}

func New() *Driver {
	d := &Driver{
		l: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})),
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

		//NewDataLog(),
		//NewFlowGetter(d.FlowRegistry),
		//NewCommiter(log),
		//NewGetter(log),
		//NewDelayer(d.l),
	}
	d.doers = doers

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

func (d *Driver) Init(e flowstate.Engine) error {
	badgerOpts := badger.DefaultOptions("")
	//badgerOpts = badgerOpts.WithLogger(badgerzlog.New(rt.l.WithPkg("badger")))
	badgerOpts = badgerOpts.WithInMemory(true)

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	d.db = db

	seq, err := db.GetSequence([]byte("rev"), 10000)
	if err != nil {
		return fmt.Errorf("get sequence: %w", err)
	}
	d.seq = seq

	// make sure we never get rev=0
	if _, err = seq.Next(); err != nil {
		return fmt.Errorf("seq: next: %w", err)
	}

	for _, doer := range d.doers {
		if err := doer.Init(e); err != nil {
			return fmt.Errorf("%T: init: %w", doer, err)
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

	if err := d.seq.Release(); err != nil {
		res = errors.Join(res, fmt.Errorf("seq: release: %w", err))
	}
	if err := d.db.Close(); err != nil {
		res = errors.Join(res, fmt.Errorf("db: close: %w", err))
	}

	return res
}
