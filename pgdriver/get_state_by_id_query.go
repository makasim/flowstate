package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

func (*queries) GetStateByID(ctx context.Context, tx conntx, id flowstate.StateID, rev int64, s *flowstate.State) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	var q string
	var qArgs []interface{}
	if rev <= 0 {
		q = `
SELECT fs.state, fs.rev 
FROM flowstate_states AS fs
INNER JOIN flowstate_latest_states AS fls 
    ON fs.id = fls.id AND fs.rev = fls.rev
WHERE fls.id = $1
LIMIT 1`
		qArgs = []interface{}{id}
	} else {
		q = `
SELECT state, rev 
FROM flowstate_states 
WHERE id = $1 AND rev = $2 
LIMIT 1`
		qArgs = []interface{}{id, rev}
	}

	if err := tx.QueryRow(ctx, q, qArgs...).Scan(s, &s.Rev); err != nil {
		return err
	}
	return nil
}
