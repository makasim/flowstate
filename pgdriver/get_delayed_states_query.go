package pgdriver

import (
	"context"
	"fmt"
)

func (*queries) GetDelayedStates(ctx context.Context, tx conntx, dm delayerMeta) ([]delayedState, error) {
	if dm.Since == 0 {
		return nil, fmt.Errorf("since is empty")
	}
	if dm.Until == 0 {
		return nil, fmt.Errorf("until is empty")
	}
	if dm.Limit == 0 {
		return nil, fmt.Errorf("limit is empty")
	}

	rows, err := tx.Query(
		ctx,
		`SELECT execute_at, state 
FROM flowstate_delayed_states 
WHERE execute_at > $1 AND execute_at <= $2
ORDER BY execute_at ASC 
LIMIT $3`,
		dm.Since,
		dm.Until,
		dm.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]delayedState, 0, dm.Limit)
	for rows.Next() {
		var ds delayedState
		if err := rows.Scan(&ds.ExecuteAt, &ds.State); err != nil {
			return nil, err
		}
		res = append(res, ds)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return res, nil
}
