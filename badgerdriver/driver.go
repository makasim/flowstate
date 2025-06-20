package badgerdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
)

type Driver struct {
	*memdriver.FlowRegistry

	db          *badger.DB
	stateRevSeq *badger.Sequence

	doers []flowstate.Doer

	l *slog.Logger
}

func New(db *badger.DB) *Driver {

	d := &Driver{
		db:           db,
		FlowRegistry: &memdriver.FlowRegistry{},

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

		memdriver.NewFlowGetter(d.FlowRegistry),
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
	stateRevSeq, err := getStateRevSequence(d.db)
	if err != nil {
		return fmt.Errorf("get state rev seq: %w", err)
	}
	d.stateRevSeq = stateRevSeq
	
	d.doers = append(d.doers, NewCommiter(d.db, d.stateRevSeq))
	for _, doer := range d.doers {
		if err := doer.Init(e); err != nil {
			return fmt.Errorf("%T: init: %w", doer, err)
		}
	}

	d.doers = append(d.doers, NewGetter(d.db))

	return nil
}

func (d *Driver) Shutdown(ctx context.Context) error {
	var res error
	for _, doer := range d.doers {
		if err := doer.Shutdown(ctx); err != nil {
			res = errors.Join(res, fmt.Errorf("%T: shutdown: %w", doer, err))
		}
	}

	if err := d.stateRevSeq.Release(); err != nil {
		res = errors.Join(res, fmt.Errorf("seq: release: %w", err))
	}
	if err := d.db.Close(); err != nil {
		res = errors.Join(res, fmt.Errorf("db: close: %w", err))
	}

	return res
}

func getStateRevSequence(db *badger.DB) (*badger.Sequence, error) {
	seq, err := db.GetSequence([]byte("rev.state"), 10000)
	if err != nil {
		return nil, err
	}
	// make sure we never get rev=0
	if _, err := seq.Next(); err != nil {
		return nil, err
	}

	return seq, nil
}
