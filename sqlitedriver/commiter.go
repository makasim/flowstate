package sqlitedriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/makasim/flowstate"
)

type Commiter struct {
	db *sql.DB

	e *flowstate.Engine
}

func NewCommiter(db *sql.DB) *Commiter {
	return &Commiter{
		db: db,
	}
}

func (d *Commiter) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*flowstate.CommitCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if len(cmd.Commands) == 0 {
		return fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*flowstate.CommitCommand); ok {
			return fmt.Errorf("commit command not allowed inside another commit")
		}
		if _, ok := c.(*flowstate.ExecuteCommand); ok {
			return fmt.Errorf("execute command not allowed inside commit")
		}
	}

	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	defer tx.Rollback()

	for _, cmd0 := range cmd.Commands {
		if err := d.e.Do(cmd0); err != nil {
			return fmt.Errorf("%T: do: %w", cmd0, err)
		}

		cmd1, ok := cmd0.(flowstate.CommittableCommand)
		if !ok {
			continue
		}

		stateCtx := cmd1.CommittableStateCtx()
		var nextRev int64
		if err = tx.QueryRow(`INSERT INTO flowstate_rev default values RETURNING rev;`).Scan(&nextRev); err != nil {
			return fmt.Errorf("db: query next rev: %w", err)
		}

		if stateCtx.Committed.Rev == 0 {
			if _, err = tx.Exec(
				`INSERT INTO flowstate_state_latest(id, rev) VALUES(?, ?)`,
				stateCtx.Current.ID,
				nextRev,
			); err != nil {
				return fmt.Errorf("db: insert latest state: %w", err)
			}
		} else {
			if _, err = tx.Exec(
				`UPDATE flowstate_state_latest SET rev = ? WHERE id = ? AND rev = ? LIMIT 1`,
				nextRev,
				stateCtx.Current.ID,
				stateCtx.Committed.Rev,
			); err != nil {
				return fmt.Errorf("db: update latest state: %w", err)
			}
		}

		serializedState, err := json.Marshal(stateCtx.Current)
		if err != nil {
			return fmt.Errorf("json: marshal state ctx: %w", err)
		}
		if _, err = tx.Exec(
			`INSERT INTO flowstate_state_log(id, rev, state) VALUES(?, ?, ?)`,
			stateCtx.Current.ID,
			nextRev,
			serializedState,
		); err != nil {
			return fmt.Errorf("db: insert log state: %w", err)
		}

	}

	return nil
}

func (d *Commiter) Init(e *flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Commiter) Shutdown(_ context.Context) error {
	return nil
}
