package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

func (*queries) GetStateByLabels(ctx context.Context, tx conntx, labels map[string]string, sinceRev int64, s *flowstate.State) error {
	if len(labels) == 0 {
		return fmt.Errorf("labels are empty")
	}

	if err := tx.QueryRow(
		ctx,
		`
SELECT state, rev 
FROM flowstate_states 
WHERE labels::JSONB @> $1 AND rev > $2
ORDER BY "rev" DESC LIMIT 1;`,
		labels,
		sinceRev,
	).Scan(s, &s.Rev); err != nil {
		return err
	}
	return nil
}
