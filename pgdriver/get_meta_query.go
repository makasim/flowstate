package pgdriver

import (
	"context"
	"errors"
)

func (*queries) GetMeta(ctx context.Context, tx conntx, key string, value any) error {
	if key == "" {
		return errors.New("key is empty")
	}
	if value == nil {
		return errors.New("value is nil")
	}

	if err := tx.QueryRow(
		ctx,
		`SELECT "value" FROM flowstate_meta WHERE "key" = $1`,
		key,
	).Scan(value); err != nil {
		return err
	}

	return nil
}
