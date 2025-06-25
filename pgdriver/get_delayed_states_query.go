package pgdriver

import (
	"context"
	"fmt"
	"time"

	"github.com/makasim/flowstate"
)

func (*queries) GetDelayedStates(ctx context.Context, tx conntx, since, until, offset int64, limit int) ([]flowstate.DelayedState, error) {
	if since == 0 {
		return nil, fmt.Errorf("since is empty")
	}
	if until == 0 {
		return nil, fmt.Errorf("until is empty")
	}
	if limit == 0 {
		return nil, fmt.Errorf("limit is empty")
	}

	rows, err := tx.Query(
		ctx,
		`
SELECT execute_at, state, pos 
FROM 
    (
		SELECT xmin::text::bigint, execute_at, state, pos 
		FROM flowstate_delayed_states 
		WHERE execute_at > $1 AND execute_at <= $2 AND pos > $3
		ORDER BY execute_at, pos ASC 
		LIMIT $4
	) AS subquery
	CROSS JOIN (
    	SELECT 
        split_part(pg_current_snapshot()::text, ':', 1)::bigint AS xmin, 
        split_part(pg_current_snapshot()::text, ':', 2)::bigint AS xmax
	) AS snapshot
WHERE subquery.xmin < snapshot.xmin OR subquery.xmin > snapshot.xmax`,
		since,
		until,
		offset,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]flowstate.DelayedState, 0, limit)
	for rows.Next() {
		var ds flowstate.DelayedState
		var executeAt int64
		if err := rows.Scan(&executeAt, &ds.State, &ds.Offset); err != nil {
			return nil, err
		}
		ds.ExecuteAt = time.Unix(executeAt, 0)

		res = append(res, ds)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return res, nil
}
