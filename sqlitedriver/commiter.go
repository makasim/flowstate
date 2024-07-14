package sqlitedriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/makasim/flowstate"
)

type Commiter struct {
	db *sql.DB
	d  *Delayer
	dl *DataLog

	e *flowstate.Engine
}

func NewCommiter(db *sql.DB, d *Delayer, dl *DataLog) *Commiter {
	return &Commiter{
		db: db,
		d:  d,
		dl: dl,
	}
}

func (d *Commiter) Do(cmd0 flowstate.Command) error {
	if _, ok := cmd0.(*flowstate.CommitStateCtxCommand); ok {
		return nil
	}

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

	conflictErr := &flowstate.ErrCommitConflict{}

	for _, subCmd0 := range cmd.Commands {
		if subCmd, ok := subCmd0.(*flowstate.DelayCommand); ok {
			if err := d.d.DoTx(tx, subCmd); err != nil {
				return fmt.Errorf("%T: do: %w", subCmd, err)
			}
		} else if subCmd, ok := subCmd0.(*flowstate.StoreDataCommand); ok {
			if err := d.dl.AddTx(tx, subCmd.Data); err != nil {
				return fmt.Errorf("%T: do: %w", subCmd, err)
			}
		} else if subCmd, ok := subCmd0.(*flowstate.GetDataCommand); ok {
			if err := d.dl.GetTx(tx, subCmd.Data); err != nil {
				return fmt.Errorf("%T: do: %w", subCmd, err)
			}
		} else {
			if err := d.e.Do(subCmd0); err != nil {
				return fmt.Errorf("%T: do: %w", subCmd0, err)
			}
		}

		cmd1, ok := subCmd0.(flowstate.CommittableCommand)
		if !ok {
			continue
		}

		nextRevRes, err := tx.Exec(`INSERT INTO flowstate_rev(rev) VALUES(NULL)`)
		if err != nil {
			return fmt.Errorf("db: query next rev: %w", err)
		}
		nextRev, err := nextRevRes.LastInsertId()
		if err != nil {
			return fmt.Errorf("db: query next rev: last insert id: %w", err)
		}

		stateCtx := cmd1.CommittableStateCtx()
		if stateCtx.Committed.Rev == 0 {
			if _, err = tx.Exec(
				`INSERT INTO flowstate_state_latest(id, rev) VALUES(?, ?)`,
				stateCtx.Current.ID,
				nextRev,
			); err != nil && strings.Contains(err.Error(), `UNIQUE constraint failed`) {
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, err)
				continue
			} else if err != nil {
				return fmt.Errorf("db: insert latest state: %w", err)
			}
		} else {
			latestStateRes, err := tx.Exec(
				`UPDATE flowstate_state_latest SET rev = ? WHERE id = ? AND rev = ?`,
				nextRev,
				stateCtx.Current.ID,
				stateCtx.Committed.Rev,
			)
			if err != nil {
				return fmt.Errorf("db: update latest state: %w", err)
			}

			affectedCnt, err := latestStateRes.RowsAffected()
			if err != nil {
				return fmt.Errorf("db: update latest state: rows affected: %w", err)
			}
			if affectedCnt == 0 {
				conflictErr.Add(fmt.Sprintf("%T", cmd), stateCtx.Current.ID, fmt.Errorf("rev mismatch"))
				continue
			}
		}

		nextState := stateCtx.Current.CopyTo(&flowstate.State{})
		nextState.SetCommitedAt(time.Now())
		nextState.Rev = nextRev

		serializedState, err := json.Marshal(nextState)
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

		nextState.CopyTo(&stateCtx.Committed)
		nextState.CopyTo(&stateCtx.Current)
		stateCtx.Transitions = stateCtx.Transitions[:0]
	}

	if len(conflictErr.TaskIDs()) > 0 {
		return conflictErr
	}

	return tx.Commit()
}

func (d *Commiter) Init(e *flowstate.Engine) error {
	d.e = e
	return nil
}

func (d *Commiter) Shutdown(_ context.Context) error {
	return nil
}
