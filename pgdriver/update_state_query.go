package pgdriver

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate"
)

func (*queries) UpdateState(ctx context.Context, tx conntx, s *flowstate.State) error {
	if s.ID == "" {
		return fmt.Errorf("id is empty")
	}
	if s.Rev == 0 {
		return fmt.Errorf("rev is empty")
	}

	if err := tx.QueryRow(
		ctx,
		`
WITH latest_state AS (
  UPDATE flowstate_latest_states SET rev = nextval('flowstate_states_rev_seq') WHERE id = @id AND rev = @curr_rev 
  RETURNING rev
)
INSERT INTO flowstate_states (rev, id, state, labels)
SELECT rev, @id, @state, @labels
FROM latest_state 
RETURNING rev
`,
		pgx.NamedArgs{
			"id":       s.ID,
			"curr_rev": s.Rev,
			"state":    s,
			"labels":   s.Labels,
		},
	).Scan(&s.Rev); err != nil {
		return err
	}

	return nil
}
