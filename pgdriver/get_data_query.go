package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

func (*queries) GetData(ctx context.Context, tx conntx, rev int64, d *flowstate.Data) error {
	if rev == 0 {
		return fmt.Errorf("rev is empty")
	}

	d.Blob = d.Blob[:0]

	if err := tx.QueryRow(
		ctx,
		`
SELECT 
    rev, 
    annotations, 
    CASE WHEN COALESCE((annotations->>'binary')::boolean, false) 
		THEN decode(b, 'base64')::bytea
        ELSE b::bytea
    END
FROM flowstate_data 
WHERE rev = $1;
`,
		rev,
	).Scan(&d.Rev, &d.Annotations, &d.Blob); err != nil {
		return err
	}
	return nil
}
