package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

func (*queries) InsertData(ctx context.Context, tx conntx, d *flowstate.Data) error {
	if d.ID == "" {
		return fmt.Errorf("id is empty")
	}
	if len(d.B) == 0 {
		return fmt.Errorf("bytes is empty")
	}

	if err := tx.QueryRow(
		ctx,
		`INSERT INTO flowstate_data(id, rev, bytes) VALUES($1, nextval('flowstate_data_rev_seq'), $2) RETURNING rev`,
		d.ID,
		d.B,
	).Scan(&d.Rev); err != nil {
		return err
	}
	return nil
}
