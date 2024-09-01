package pgdriver

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/flowstate"
)

func (*queries) InsertDelayedState(ctx context.Context, tx conntx, s flowstate.State, executeAt time.Time) error {
	if s.ID == "" {
		return fmt.Errorf("id is empty")
	}
	if executeAt.IsZero() {
		return fmt.Errorf("execute at is zero")
	}

	res, err := tx.Exec(
		ctx,
		`INSERT INTO flowstate_delayed_states(execute_at, state) VALUES($1, $2)`,
		executeAt.Unix(),
		s,
	)
	if err != nil {
		return fmt.Errorf("db: insert delay log: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected")
	}

	return nil
}
