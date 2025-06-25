package pgdriver

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Delayer{}

type delayerQueries interface {
	InsertDelayedState(ctx context.Context, tx conntx, s flowstate.State, executeAt time.Time) error
	GetDelayedStates(ctx context.Context, tx conntx, since, until, offset int64, limit int) ([]flowstate.DelayedState, error)
}

type Delayer struct {
	conn   conn
	q      delayerQueries
	now    func() time.Time
	doneCh chan struct{}

	e flowstate.Engine
}

func NewDelayer(conn conn, q delayerQueries, now func() time.Time) *Delayer {
	return &Delayer{
		conn: conn,
		q:    q,
		now:  now,

		doneCh: make(chan struct{}),
	}
}

func (d *Delayer) Do(cmd0 flowstate.Command) error {
	switch cmd := cmd0.(type) {
	case *flowstate.DelayCommand:
		if err := cmd.Prepare(); err != nil {
			return err
		}

		executeAt := time.Now().Add(cmd.Duration)

		if err := d.q.InsertDelayedState(context.Background(), d.conn, cmd.DelayStateCtx.Current, executeAt); err != nil {
			return fmt.Errorf("insert delayed state query: %w", err)
		}

		return nil
	case *flowstate.GetDelayedStatesCommand:
		cmd.Prepare()

		ds, err := d.q.GetDelayedStates(context.Background(), d.conn, cmd.Since.Unix(), cmd.Until.Unix(), cmd.Offset, cmd.Limit+1)
		if err != nil {
			return fmt.Errorf("get delayed states query: %w", err)
		}

		var more bool
		if len(ds) > cmd.Limit {
			ds = ds[:cmd.Limit]
			more = true
		}

		cmd.SetResult(&flowstate.GetDelayedStatesResult{
			States: ds,
			More:   more,
		})

		return nil
	default:
		return flowstate.ErrCommandNotSupported
	}
}

func (d *Delayer) Init(e flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Delayer) Shutdown(_ context.Context) error {
	select {
	case <-d.doneCh:
		return fmt.Errorf(`already shutdown`)
	default:
		close(d.doneCh)
		return nil
	}
}
