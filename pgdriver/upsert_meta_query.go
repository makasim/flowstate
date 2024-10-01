package pgdriver

import (
	"context"
	"errors"
	"fmt"
)

func (*queries) UpsertMeta(ctx context.Context, tx conntx, key string, value any) error {
	if key == "" {
		return errors.New("key is empty")
	}

	res, err := tx.Exec(
		ctx,
		`
INSERT INTO flowstate_meta("key", "value") VALUES($1, $2) 
ON CONFLICT ("key") DO 
UPDATE SET value = $2`,
		key,
		value,
	)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("no rows affected")
	}

	return nil
}
