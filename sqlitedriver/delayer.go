package sqlitedriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Delayer{}

type Delayer struct {
	db     *sql.DB
	doneCh chan struct{}

	e *flowstate.Engine
}

func NewDelayer(db *sql.DB) *Delayer {
	return &Delayer{
		db:     db,
		doneCh: make(chan struct{}),
	}
}

func (d *Delayer) Do(cmd0 flowstate.Command) error {
	if _, ok := cmd0.(*delayedCommit); ok {
		return nil
	}

	cmd, ok := cmd0.(*flowstate.DelayCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	if err := cmd.Prepare(); err != nil {
		return err
	}

	executeAt := time.Now().Add(cmd.Duration)

	serializedState, err := json.Marshal(cmd.DelayStateCtx)
	if err != nil {
		return fmt.Errorf("json: marshal state ctx: %w", err)
	}
	if _, err = d.db.Exec(
		`INSERT INTO flowstate_delay_log(state, execute_at) VALUES(?, ?)`,
		serializedState,
		executeAt.Unix(),
	); err != nil {
		return fmt.Errorf("db: insert delay log: %w", err)
	}

	return nil
}

func (d *Delayer) DoTx(tx *sql.Tx, cmd *flowstate.DelayCommand) error {
	if err := cmd.Prepare(); err != nil {
		return err
	}

	executeAt := time.Now().Add(cmd.Duration)

	serializedState, err := json.Marshal(cmd.DelayStateCtx)
	if err != nil {
		return fmt.Errorf("json: marshal state ctx: %w", err)
	}
	if _, err = tx.Exec(
		`INSERT INTO flowstate_delay_log(state, execute_at) VALUES(?, ?)`,
		serializedState,
		executeAt.Unix(),
	); err != nil {
		return fmt.Errorf("db: insert delay log: %w", err)
	}

	return nil
}

func (d *Delayer) Init(e *flowstate.Engine) error {
	if _, err := d.db.Exec(createDelayLogTableSQL); err != nil {
		return fmt.Errorf("create flowstate_delay_log table: db: exec: %w", err)
	}
	if _, err := d.db.Exec(createDelayMeta); err != nil {
		return fmt.Errorf("create flowstate_delay_meta table: db: exec: %w", err)
	}

	d.e = e

	go func() {
		t := time.NewTicker(time.Millisecond * 100)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				if err := d.do(); err != nil {
					log.Printf(`ERROR: sqlitedriver: delayer: do: %s`, err)
				}
			case <-d.doneCh:
				return
			}
		}
	}()

	return nil
}

func (d *Delayer) do() error {
	executeUntilQuery := `
SELECT executed_until 
FROM flowstate_delay_meta 
ORDER BY executed_until DESC 
LIMIT 1`

	delayedStatesQuery := `
SELECT execute_at, state 
FROM flowstate_delay_log 
WHERE execute_at > ? 
ORDER BY execute_at ASC 
LIMIT 10`

	var executedUntil int64
	if err := d.db.QueryRow(executeUntilQuery).Scan(&executedUntil); errors.Is(err, sql.ErrNoRows) {
		// ok
	} else if err != nil {
		return fmt.Errorf(`query executed until: %w`, err)
	}

	rows, err := d.db.Query(delayedStatesQuery, executedUntil)
	if err != nil {
		return fmt.Errorf(`query delayed states: %w`, err)
	}
	defer rows.Close()

	var stateCtxJSON []byte
	var nextExecutedAt int64

	var stateCtxs []*flowstate.StateCtx

	for rows.Next() {
		stateCtxJSON = stateCtxJSON[:0]

		if err := rows.Scan(&nextExecutedAt, &stateCtxJSON); err != nil {
			return fmt.Errorf(`rows: scan delayed state: %w`, err)
		}

		stateCtx := &flowstate.StateCtx{}
		if err := json.Unmarshal(stateCtxJSON, stateCtx); err != nil {
			return fmt.Errorf(`unmarshal delayed state ctx: %w`, err)
		}

		stateCtxs = append(stateCtxs, stateCtx)
	}
	if rows.Err() != nil {
		return fmt.Errorf(`rows: next: %w`, rows.Err())
	}
	rows.Close()

	for _, stateCtx := range stateCtxs {
		if stateCtx.Current.Transition.Annotations[flowstate.DelayCommitAnnotation] == `true` {
			conflictErr := &flowstate.ErrCommitConflict{}
			if err := d.e.Do(flowstate.Commit(&delayedCommit{stateCtx: stateCtx})); errors.As(err, conflictErr) {
				log.Printf("ERROR: engine: commit: %s\n", conflictErr)
				continue
			} else if err != nil {
				return fmt.Errorf("engine: commit: %w", err)
			}
		}

		go func() {
			if err := d.e.Execute(stateCtx); err != nil {
				log.Printf(`ERROR: sqlitedriver: delayer: engine: execute: %s`, err)
			}
		}()
	}

	if _, err := d.db.Exec(`INSERT INTO flowstate_delay_meta(executed_until) VALUES(?)`, nextExecutedAt); err != nil {
		return fmt.Errorf(`query executed until: %w`, err)
	}

	return nil
}

func (d *Delayer) Shutdown(_ context.Context) error {
	close(d.doneCh)

	return nil
}

type delayedCommit struct {
	stateCtx *flowstate.StateCtx
}

func (cmd *delayedCommit) CommittableStateCtx() *flowstate.StateCtx {
	return cmd.stateCtx
}

func (cmd *delayedCommit) Prepare() error {
	return nil
}
