package pgdriver

import (
	"context"

	"github.com/makasim/flowstate"
)

func (*queries) InsertData(ctx context.Context, tx conntx, d *flowstate.Data) error {
	if err := tx.QueryRow(
		ctx,
		`
INSERT INTO flowstate_data(rev, annotations, b)
VALUES(
	nextval('flowstate_data_rev_seq'), 
	$1, 
	CASE WHEN $2 
		THEN encode($3, 'base64') 
		ELSE $3::text
	END
) RETURNING rev`,
		d.Annotations,
		d.IsBinary(),
		d.Blob,
	).Scan(&d.Rev); err != nil {
		return err
	}
	return nil
}
