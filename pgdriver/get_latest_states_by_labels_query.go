package pgdriver

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
)

func (*queries) GetLatestStatesByLabels(ctx context.Context, tx conntx, orLabels []map[string]string, sinceRev int64, sinceTime time.Time, ss []flowstate.State) ([]flowstate.State, error) {
	fromStmt := `FROM flowstate_latest_states AS latest_states 
	INNER JOIN flowstate_states AS states ON latest_states.id = states.id AND latest_states.rev = states.rev`

	return getStatesByLabelsWithFromStatement(ctx, tx, fromStmt, orLabels, sinceRev, sinceTime, ss)
}
