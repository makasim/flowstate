package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/makasim/flowstate"
)

type Driver struct {
	*flowstate.FlowRegistry
	conn  conn
	q     *queries
	doers []flowstate.Driver

	recoverer flowstate.Driver
	l         *slog.Logger
}

func New(conn conn, l *slog.Logger) *Driver {
	return &Driver{
		conn: conn,

		q:            &queries{},
		FlowRegistry: &flowstate.FlowRegistry{},
		l:            l,
	}
}

func (d *Driver) GetData(cmd *flowstate.GetDataCommand) error {
	if err := d.q.GetData(context.Background(), d.conn, cmd.Data.ID, cmd.Data.Rev, cmd.Data); err != nil {
		return fmt.Errorf("get data query: %w", err)
	}

	return nil
}

func (d *Driver) StoreData(cmd *flowstate.StoreDataCommand) error {
	if err := d.q.InsertData(context.Background(), d.conn, cmd.Data); err != nil {
		return fmt.Errorf("insert data queue: %w", err)
	}

	return nil
}

func (d *Driver) GetStateByID(cmd *flowstate.GetStateByIDCommand) error {
	s := flowstate.State{}
	if err := d.q.GetStateByID(context.Background(), d.conn, cmd.ID, cmd.Rev, &s); errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w; id=%v rev=%d", flowstate.ErrNotFound, cmd.ID, cmd.Rev)
	} else if err != nil {
		return fmt.Errorf("get state by id: query: %w", err)
	}

	s.CopyToCtx(cmd.StateCtx)
	return nil
}

func (d *Driver) GetStateByLabels(cmd *flowstate.GetStateByLabelsCommand) error {
	s := flowstate.State{}

	ss := []flowstate.State{s}
	ss, err := d.q.GetStatesByLabels(context.Background(), d.conn, []map[string]string{cmd.Labels}, -1, time.Time{}, ss)
	if err != nil {
		return fmt.Errorf("get states by labels query: %w", err)
	} else if len(ss) == 0 {
		return fmt.Errorf("%w; labels=%v", flowstate.ErrNotFound, cmd.Labels)
	}
	s = ss[0]

	s.CopyToCtx(cmd.StateCtx)
	return nil
}

func (d *Driver) GetStates(cmd *flowstate.GetStatesCommand) (*flowstate.GetStatesResult, error) {
	ss := make([]flowstate.State, cmd.Limit+1)
	if cmd.LatestOnly {
		var err error
		ss, err = d.q.GetLatestStatesByLabels(context.Background(), d.conn, cmd.Labels, cmd.SinceRev, cmd.SinceTime, ss)
		if err != nil {
			return nil, fmt.Errorf("get latest states by labels query: %w", err)
		}
	} else {
		var err error
		ss, err = d.q.GetStatesByLabels(context.Background(), d.conn, cmd.Labels, cmd.SinceRev, cmd.SinceTime, ss)
		if err != nil {
			return nil, fmt.Errorf("get states by labels query: %w", err)
		}
	}

	var more bool
	if len(ss) > cmd.Limit {
		ss = ss[:cmd.Limit]
		more = true
	}

	return &flowstate.GetStatesResult{
		States: ss,
		More:   more,
	}, nil
}

func (d *Driver) Delay(cmd *flowstate.DelayCommand) error {
	if err := d.q.InsertDelayedState(context.Background(), d.conn, cmd.DelayingState, cmd.ExecuteAt); err != nil {
		return fmt.Errorf("insert delayed state query: %w", err)
	}

	return nil
}

func (d *Driver) GetDelayedStates(cmd *flowstate.GetDelayedStatesCommand) (*flowstate.GetDelayedStatesResult, error) {
	ds, err := d.q.GetDelayedStates(context.Background(), d.conn, cmd.Since.Unix(), cmd.Until.Unix(), cmd.Offset, cmd.Limit+1)
	if err != nil {
		return nil, fmt.Errorf("get delayed states query: %w", err)
	}

	var more bool
	if len(ds) > cmd.Limit {
		ds = ds[:cmd.Limit]
		more = true
	}

	return &flowstate.GetDelayedStatesResult{
		States: ds,
		More:   more,
	}, nil
}

func (d *Driver) Commit(cmd *flowstate.CommitCommand, e flowstate.Engine) error {
	tx, err := d.conn.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("conn: begin: %w", err)
	}
	defer tx.Rollback(context.Background())

	revMismatchErr := &flowstate.ErrRevMismatch{}

	for _, subCmd0 := range cmd.Commands {
		if _, ok := subCmd0.(*flowstate.CommitStateCtxCommand); !ok {
			if err := e.Do(subCmd0); err != nil {
				return fmt.Errorf("%T: do: %w", subCmd0, err)
			}
		}

		committableCmd, ok := subCmd0.(flowstate.CommittableCommand)
		if !ok {
			continue
		}
		committableStateCtx := committableCmd.CommittableStateCtx()

		nextState := committableStateCtx.Current.CopyTo(&flowstate.State{})
		nextState.SetCommitedAt(time.Now())

		if committableStateCtx.Committed.Rev > 0 {
			if err := d.q.UpdateState(context.Background(), tx, &nextState); isRevMismatchErr(err) {
				revMismatchErr.Add(fmt.Sprintf("%T", cmd), committableStateCtx.Current.ID, err)
				return revMismatchErr
			} else if err != nil {
				return fmt.Errorf("update state: %w", err)
			}
		} else {
			if err := d.q.InsertState(context.Background(), tx, &nextState); isRevMismatchErr(err) {
				revMismatchErr.Add(fmt.Sprintf("%T", cmd), committableStateCtx.Current.ID, err)
				return revMismatchErr
			} else if err != nil {
				return fmt.Errorf("insert state: %w", err)
			}
		}

		nextState.CopyTo(&committableStateCtx.Committed)
		nextState.CopyTo(&committableStateCtx.Current)
		committableStateCtx.Transitions = committableStateCtx.Transitions[:0]
	}

	if len(revMismatchErr.TaskIDs()) > 0 {
		return revMismatchErr
	}

	return tx.Commit(context.Background())
}

func (d *Driver) GetFlow(cmd *flowstate.GetFlowCommand) error {
	return d.FlowRegistry.Do(cmd)
}

func isRevMismatchErr(err error) bool {
	if errors.Is(err, pgx.ErrNoRows) {
		return true
	}

	pgErr := &pgconn.PgError{}
	if !errors.As(err, &pgErr) {
		return false
	}

	uniqueViolationCode := `23505`
	return pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == `flowstate_latest_states_pkey`
}
