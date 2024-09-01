package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

func (*queries) GetData(ctx context.Context, tx conntx, d *flowstate.Data) error {
	if d.ID == "" {
		return fmt.Errorf("id is empty")
	}
	if d.Rev == 0 {
		return fmt.Errorf("rev is empty")
	}

	d.B = d.B[:0]

	if err := tx.QueryRow(
		ctx,
		`SELECT bytes FROM flowstate_data WHERE id = $1 AND rev = $2`,
		d.ID,
		d.Rev,
	).Scan(&d.B); err != nil {
		return err
	}
	return nil
}
