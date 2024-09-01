package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

func (*queries) GetData(ctx context.Context, tx conntx, id flowstate.DataID, rev int64, d *flowstate.Data) error {
	if id == "" {
		return fmt.Errorf("id is empty")
	}
	if rev == 0 {
		return fmt.Errorf("rev is empty")
	}

	d.B = d.B[:0]

	if err := tx.QueryRow(
		ctx,
		`SELECT id, rev, bytes FROM flowstate_data WHERE id = $1 AND rev = $2`,
		id,
		rev,
	).Scan(&d.ID, &d.Rev, &d.B); err != nil {
		return err
	}
	return nil
}
