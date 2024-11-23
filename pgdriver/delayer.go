package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Delayer{}

type delayerQueries interface {
	InsertDelayedState(ctx context.Context, tx conntx, s flowstate.State, executeAt time.Time) error
	GetDelayedStates(ctx context.Context, tx conntx, dm delayerMeta) ([]delayedState, error)
	GetMeta(ctx context.Context, tx conntx, key string, value any) error
	UpsertMeta(ctx context.Context, tx conntx, key string, value any) error
}

type delayerMeta struct {
	Limit int   `json:"limit"`
	Pos   int64 `json:"pos"`
	Since int64 `json:"since"`
	Until int64 `json:"-"`
}

type delayedState struct {
	ExecuteAt int64
	Pos       int64
	State     flowstate.State
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
	cmd, ok := cmd0.(*flowstate.DelayCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if err := cmd.Prepare(); err != nil {
		return err
	}

	executeAt := time.Now().Add(cmd.Duration)

	if err := d.q.InsertDelayedState(context.Background(), d.conn, cmd.DelayStateCtx.Current, executeAt); err != nil {
		return fmt.Errorf("insert delayed state query: %w", err)
	}

	return nil
}

func (d *Delayer) Init(e flowstate.Engine) error {
	d.e = e

	go func() {
		t := time.NewTicker(time.Millisecond * 100)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				if err := d.do(); err != nil {
					log.Printf(`ERROR: sqlitedriver: delayer: do: %s`, err)
				}
			case <-d.doneCh:
				return
			}
		}
	}()

	return nil
}

func (d *Delayer) do() error {
	dm := delayerMeta{}
	if err := d.q.GetMeta(context.Background(), d.conn, d.key(), &dm); errors.Is(err, pgx.ErrNoRows) {
		dm = delayerMeta{
			Limit: 100,
			Since: 1,
			Pos:   0,
		}
	} else if err != nil {
		return fmt.Errorf(`get delayer meta query: %w`, err)
	}

	dm.Until = d.now().Unix()

	dss, err := d.q.GetDelayedStates(context.Background(), d.conn, dm)
	if err != nil {
		return fmt.Errorf(`query delayed states: %w`, err)
	}
	if len(dss) == 0 {
		return nil
	}

	for _, ds := range dss {
		stateCtx := ds.State.CopyToCtx(&flowstate.StateCtx{})
		if stateCtx.Current.Transition.Annotations[flowstate.DelayCommitAnnotation] == `true` {
			conflictErr := &flowstate.ErrCommitConflict{}
			if err := d.e.Do(flowstate.Commit(
				flowstate.CommitStateCtx(stateCtx),
			)); errors.As(err, conflictErr) {
				log.Printf("ERROR: engine: commit: %s\n", conflictErr)
				continue
			} else if err != nil {
				return fmt.Errorf("engine: commit: %w", err)
			}
		}

		go func() {
			if err := d.e.Execute(stateCtx); err != nil {
				log.Printf(`ERROR: pgdriver: delayer: engine: execute: %s`, err)
			}
		}()

		dm.Since = ds.ExecuteAt
		dm.Pos = ds.Pos
	}

	if err := d.q.UpsertMeta(context.Background(), d.conn, d.key(), dm); err != nil {
		return fmt.Errorf(`upsert delayer meta query: %w`, err)
	}

	return nil
}

func (*Delayer) key() string {
	return `delayer.0.meta`
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
