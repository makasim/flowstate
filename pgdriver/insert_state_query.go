package pgdriver

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate"
)

func (*queries) InsertState(ctx context.Context, tx conntx, s *flowstate.State) error {
	if s.ID == "" {
		return fmt.Errorf("id is empty")
	}
	if s.Rev != 0 {
		return fmt.Errorf("rev is not empty")
	}

	if err := tx.QueryRow(
		ctx,
		`
WITH latest_state AS (
  INSERT INTO flowstate_latest_states (rev, id) 
  VALUES (nextval('flowstate_states_rev_seq'), @id)
  RETURNING rev
)
INSERT INTO flowstate_states (rev, id, state, labels)
SELECT rev, @id, @state, @labels
FROM latest_state 
RETURNING rev
`,
		pgx.NamedArgs{
			"id":     s.ID,
			"state":  *s,
			"labels": s.Labels,
		},
	).Scan(&s.Rev); err != nil {
		return err
	}

	return nil
}
