package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	_ "github.com/mattn/go-sqlite3"
)

type Driver struct {
	*memdriver.FlowRegistry
	conn  conn
	q     *queries
	doers []flowstate.Doer
}

func New(conn conn) *Driver {
	d := &Driver{
		conn: conn,

		q:            &queries{},
		FlowRegistry: &memdriver.FlowRegistry{},
	}

	d.doers = []flowstate.Doer{
		flowstate.DefaultTransitDoer,
		flowstate.DefaultPauseDoer,
		flowstate.DefaultResumeDoer,
		flowstate.DefaultEndDoer,
		flowstate.DefaultNoopDoer,
		flowstate.DefaultSerializerDoer,
		flowstate.DefaultDeserializeDoer,
		flowstate.DefaultDereferenceDataDoer,
		flowstate.DefaultReferenceDataDoer,
		flowstate.DefaultReferenceDataDoer,
		flowstate.DefaultDereferenceDataDoer,

		flowstate.Recoverer(time.Millisecond * 500),

		memdriver.NewFlowGetter(d.FlowRegistry),
		NewDataer(d.conn, d.q),
		NewCommiter(d.conn, d.q),
		NewGetter(d.conn, d.q),

		NewWatcher(d.conn, d.q),
		NewDelayer(d.conn, d.q, time.Now),
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
	//if _, err := d.db.Exec(createRevTableSQL); err != nil {
	//	return fmt.Errorf("create flowstate_rev table: db: exec: %w", err)
	//}
	//if _, err := d.db.Exec(createStateLatestTableSQL); err != nil {
	//	return fmt.Errorf("create flowstate_state_latest table: db: exec: %w", err)
	//}
	//if _, err := d.db.Exec(createStateLogTableSQL); err != nil {
	//	return fmt.Errorf("create flowstate_state_log table: db: exec: %w", err)
	//}
	//if _, err := d.db.Exec(createDataLogTableSQL); err != nil {
	//	return fmt.Errorf("create flowstate_data_log table: db: exec: %w", err)
	//}

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

	return res
}
