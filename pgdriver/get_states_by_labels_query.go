package pgdriver

import (
	"context"
	"fmt"
	"strconv"

	"github.com/makasim/flowstate"
)

func (*queries) GetStatesByLabels(ctx context.Context, tx conntx, orLabels []map[string]string, sinceRev int64, ss []flowstate.State) ([]flowstate.State, error) {
	if len(ss) == 0 {
		return nil, fmt.Errorf("states slice len must be greater than 0")
	}

	var args []any
	var labelsWhere string
	var where string
	for i := range orLabels {
		if i > 0 {
			labelsWhere += " OR "
		}
		args = append(args, orLabels[i])
		labelsWhere += "labels::JSONB @> $" + strconv.Itoa(len(args))
	}
	if labelsWhere != "" {
		where = "(" + labelsWhere + ")"
	}

	if sinceRev >= 0 {
		if where != "" {
			where += " AND "
		}
		args = append(args, sinceRev)
		where += "rev > $" + strconv.Itoa(len(args))
	} else { // negative rev is treated as since latest
		if where != "" {
			where += " AND "
		}
		var subWhere string
		if labelsWhere != "" {
			subWhere = " WHERE (" + labelsWhere + ")"
		}

		where += `rev >= (SELECT rev FROM flowstate_states ` + subWhere + ` ORDER BY "rev" DESC LIMIT 1)`
	}

	q := `
SELECT state, rev 
FROM flowstate_states 
WHERE ` + where + `
ORDER BY "rev" DESC LIMIT ` + strconv.Itoa(len(ss)) + `;`

	rows, err := tx.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var i int
	for rows.Next() && i < len(ss) {
		if err := rows.Scan(&ss[i], &ss[i].Rev); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		i++
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows: %w", rows.Err())
	}

	return ss[:i], nil
}
