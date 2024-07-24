package sqlitedriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Getter{}

type Getter struct {
	db *sql.DB
}

func NewGetter(db *sql.DB) *Getter {
	return &Getter{
		db: db,
	}
}

func (d *Getter) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*flowstate.GetCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if len(cmd.Labels) > 0 {
		var labelsWhere string
		var args []any
		for k, v := range cmd.Labels {
			if labelsWhere != "" {
				labelsWhere += " AND "
			}

			// TODO: somewhat ? does not work inside josn_extract, sanitize k to prevent sql injection
			labelsWhere += `json_extract(state, '$.labels.` + k + `') = ?`
			args = append(args, v)
		}

		s, err := d.find(fmt.Sprintf(`SELECT state FROM flowstate_state_log WHERE %s ORDER BY rev DESC LIMIT 1`, labelsWhere), args...)
		if err != nil {
			return err
		}
		s.CopyToCtx(cmd.StateCtx)
	} else if cmd.ID != "" && cmd.Rev == 0 {
		s, err := d.find(`SELECT state FROM flowstate_state_log WHERE id = ? ORDER BY rev DESC LIMIT 1`, cmd.ID)
		if err != nil {
			return err
		}
		s.CopyToCtx(cmd.StateCtx)
	} else if cmd.ID != "" && cmd.Rev > 0 {
		s, err := d.find(`SELECT state FROM flowstate_state_log WHERE id = ? AND rev = ? ORDER BY rev DESC LIMIT 1`, cmd.ID, cmd.Rev)
		if err != nil {
			return err
		}
		s.CopyToCtx(cmd.StateCtx)
	} else {
		return fmt.Errorf("invalid get command")
	}

	return nil
}

func (d *Getter) Init(e *flowstate.Engine) error {
	return nil
}

func (d *Getter) Shutdown(_ context.Context) error {
	return nil
}

func (d *Getter) find(query string, args ...any) (flowstate.State, error) {
	var stateJSON []byte
	if err := d.db.QueryRow(query, args...).Scan(&stateJSON); err != nil {
		return flowstate.State{}, err
	}

	var state flowstate.State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return flowstate.State{}, err
	}

	return state, nil
}
