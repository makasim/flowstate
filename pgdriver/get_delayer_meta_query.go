package pgdriver

import (
	"context"
)

func (*queries) GetDelayerMeta(ctx context.Context, tx conntx, shard int, dm *delayerMeta) error {
	if err := tx.QueryRow(
		ctx,
		`SELECT meta FROM flowstate_delayer_meta WHERE shard = $1`,
		shard,
	).Scan(dm); err != nil {
		return err
	}

	return nil
}
