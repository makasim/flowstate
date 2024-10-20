package pgdriver_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/sqlitedriver"
	"github.com/stretchr/testify/require"
)

func TestDriver_Init(t *testing.T) {
	db, err := sql.Open("sqlite3", `:memory:`)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	defer db.Close()

	// driver init\shutdown performed by engine in NewEngine and e.Shutdown

	d := sqlitedriver.New(db)

	e, err := flowstate.NewEngine(d)

	require.NoError(t, e.Shutdown(context.Background()))
}

func TestDriver_Commit(main *testing.T) {
	setUp := func(t *testing.T) (*sqlitedriver.Driver, *sql.DB) {
		db, err := sql.Open("sqlite3", `:memory:`)
		require.NoError(t, err)
		db.SetMaxOpenConns(1)

		d := sqlitedriver.New(db)

		t.Cleanup(func() {
			require.NoError(t, d.Shutdown(context.Background()))
			require.NoError(t, db.Close())
		})

		_, err = flowstate.NewEngine(d)
		require.NoError(t, err)

		return d, db
	}

	main.Run("Insert", func(t *testing.T) {
		d, db := setUp(t)

		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "fooID",
				Rev: 0, // insert
			},
		}

		err := d.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `aFlowID`),
		))
		require.NoError(t, err)

		require.Equal(t, int64(1), stateCtx.Current.Rev)
		require.Equal(t, flowstate.StateID(`fooID`), stateCtx.Committed.ID)
		require.Equal(t, int64(1), stateCtx.Committed.Rev)

		var cntLatest int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM flowstate_state_latest`).Scan(&cntLatest))
		require.Equal(t, 1, cntLatest)

		var cntLog int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM flowstate_state_log`).Scan(&cntLog))
		require.Equal(t, 1, cntLog)
	})

	main.Run("InsertAlreadyExist", func(t *testing.T) {
		d, db := setUp(t)

		_, err := db.Exec(
			`INSERT INTO flowstate_state_latest(id, rev) VALUES(?, ?)`,
			`fooID`,
			123,
		)

		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "fooID",
				Rev: 0, // insert
			},
		}

		err = d.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `aFlowID`),
		))
		require.EqualError(t, err, `*sqlitedriver.Commiter: do: conflict; cmd: *flowstate.CommitCommand sid: fooID; err: UNIQUE constraint failed: flowstate_state_latest.id;`)
		require.True(t, errors.As(err, &flowstate.ErrCommitConflict{}))
	})

	main.Run("Update", func(t *testing.T) {
		d, db := setUp(t)

		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "fooID",
				Rev: 0, // insert
			},
		}

		// insert
		err := d.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `aFlowID`),
		))
		require.NoError(t, err)

		// update
		err = d.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `aFlowID`),
		))
		require.NoError(t, err)

		require.Equal(t, int64(2), stateCtx.Current.Rev)
		require.Equal(t, flowstate.StateID(`fooID`), stateCtx.Committed.ID)
		require.Equal(t, int64(2), stateCtx.Committed.Rev)

		var cntLatest int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM flowstate_state_latest`).Scan(&cntLatest))
		require.Equal(t, 1, cntLatest)

		var cntLog int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM flowstate_state_log`).Scan(&cntLog))
		require.Equal(t, 2, cntLog)
	})

	main.Run("UpdateConflict", func(t *testing.T) {
		d, db := setUp(t)

		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "fooID",
				Rev: 0, // insert
			},
		}

		// insert
		err := d.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `aFlowID`),
		))
		require.NoError(t, err)

		stateCtx.Committed.Rev = 123

		// update
		err = d.Do(flowstate.Commit(
			flowstate.Transit(stateCtx, `aFlowID`),
		))
		require.EqualError(t, err, `*sqlitedriver.Commiter: do: conflict; cmd: *flowstate.CommitCommand sid: fooID; err: rev mismatch;`)
		require.True(t, errors.As(err, &flowstate.ErrCommitConflict{}))

		var cntLatest int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM flowstate_state_latest`).Scan(&cntLatest))
		require.Equal(t, 1, cntLatest)

		var cntLog int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM flowstate_state_log`).Scan(&cntLog))
		require.Equal(t, 1, cntLog)
	})

}
