package pgdriver

import (
	"context"
	"fmt"
)

func (*queries) UpsertDelayerMeta(ctx context.Context, tx conntx, dm delayerMeta) error {
	res, err := tx.Exec(
		ctx,
		`
INSERT INTO flowstate_delayer_meta(shard, meta) VALUES($1, $2) 
ON CONFLICT (shard) DO 
UPDATE SET meta = $2`,
		dm.Shard,
		dm,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected")
	}

	return nil
}
