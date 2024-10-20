package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/makasim/flowstate"
)

type commiterQueries interface {
	InsertState(ctx context.Context, tx conntx, s *flowstate.State) error
	UpdateState(ctx context.Context, tx conntx, s *flowstate.State) error
	InsertData(ctx context.Context, tx conntx, d *flowstate.Data) error
	InsertDelayedState(ctx context.Context, tx conntx, s flowstate.State, executeAt time.Time) error
}

type Commiter struct {
	conn conn
	q    commiterQueries
	e    *flowstate.Engine
}

func NewCommiter(conn conn, db commiterQueries) *Commiter {
	return &Commiter{
		conn: conn,
		q:    db,
	}
}

func (cmtr *Commiter) Do(cmd0 flowstate.Command) error {
	if _, ok := cmd0.(*flowstate.CommitStateCtxCommand); ok {
		return nil
	}

	cmd, ok := cmd0.(*flowstate.CommitCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if len(cmd.Commands) == 0 {
		return fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*flowstate.CommitCommand); ok {
			return fmt.Errorf("commit command not allowed inside another commit")
		}
		if _, ok := c.(*flowstate.ExecuteCommand); ok {
			return fmt.Errorf("execute command not allowed inside commit")
		}
	}

	tx, err := cmtr.conn.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("conn: begin: %w", err)
	}
	defer tx.Rollback(context.Background())

	conflictErr := &flowstate.ErrCommitConflict{}

	for _, subCmd0 := range cmd.Commands {
		if subCmd, ok := subCmd0.(*flowstate.DelayCommand); ok {
			if err := subCmd.Prepare(); err != nil {
				return fmt.Errorf("%T: prepare: %w", subCmd, err)
			}

			executeAt := time.Now().Add(subCmd.Duration)
			if err := cmtr.q.InsertDelayedState(context.Background(), tx, subCmd.DelayStateCtx.Current, executeAt); err != nil {
				return fmt.Errorf("%T: do: %w", subCmd, err)
			}
		} else if subCmd, ok := subCmd0.(*flowstate.StoreDataCommand); ok {
			if err := subCmd.Prepare(); err != nil {
				return fmt.Errorf("%T: prepare: %w", subCmd, err)
			}
			if err := cmtr.q.InsertData(context.Background(), tx, subCmd.Data); err != nil {
				return fmt.Errorf("%T: do: %w", subCmd, err)
			}
		} else {
			if err := cmtr.e.Do(subCmd0); err != nil {
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
			if err := cmtr.q.UpdateState(context.Background(), tx, &nextState); isConflict(err) {
				conflictErr.Add(fmt.Sprintf("%T", cmd), committableStateCtx.Current.ID, err)
				continue
			} else if err != nil {
				return fmt.Errorf("update state: %w", err)
			}
		} else {
			if err := cmtr.q.InsertState(context.Background(), tx, &nextState); isConflict(err) {
				conflictErr.Add(fmt.Sprintf("%T", cmd), committableStateCtx.Current.ID, err)
				continue
			} else if err != nil {
				return fmt.Errorf("insert state: %w", err)
			}
		}

		nextState.CopyTo(&committableStateCtx.Committed)
		nextState.CopyTo(&committableStateCtx.Current)
		committableStateCtx.Transitions = committableStateCtx.Transitions[:0]
	}

	if len(conflictErr.TaskIDs()) > 0 {
		return conflictErr
	}

	return tx.Commit(context.Background())
}

func (cmtr *Commiter) Init(e *flowstate.Engine) error {
	cmtr.e = e
	return nil
}

func (cmtr *Commiter) Shutdown(_ context.Context) error {
	return nil
}

func isConflict(err error) bool {
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
