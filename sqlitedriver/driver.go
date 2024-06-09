package sqlitedriver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/exptcmd"
	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/stddoer"

	_ "github.com/mattn/go-sqlite3"
)

type Driver struct {
	*memdriver.FlowRegistry
	db    *sql.DB
	doers []flowstate.Doer
}

func New(db *sql.DB) *Driver {
	d := &Driver{
		FlowRegistry: &memdriver.FlowRegistry{},

		db: db,
	}

	doers := []flowstate.Doer{
		stddoer.Transit(),
		stddoer.Pause(),
		stddoer.Resume(),
		stddoer.End(),
		stddoer.Noop(),

		exptcmd.ForkDoer(),
		exptcmd.NewStacker(),
		exptcmd.UnstackDoer(),

		memdriver.NewFlowGetter(d.FlowRegistry),

		NewCommiter(d.db),
		NewWatcher(d.db),

		// TODO: implement sqlite doers and remove the following doers
		//memdriver.NewDelayer(),
	}
	d.doers = doers

	return d
}

func (d *Driver) Do(cmd0 flowstate.Command) error {
	if cmd, ok := cmd0.(*flowstate.GetFlowCommand); ok {
		return d.doGetFlow(cmd)
	}

	for _, doer := range d.doers {
		if err := doer.Do(cmd0); errors.Is(err, flowstate.ErrCommandNotSupported) {
			continue
		} else if err != nil {
			return fmt.Errorf("%T: do: %w", doer, err)
		}

		return nil
	}

	return fmt.Errorf("no doer for command %T", cmd0)
}

func (d *Driver) Init(e *flowstate.Engine) error {
	if _, err := d.db.Exec(createRevTableSQL); err != nil {
		return fmt.Errorf("create flowstate_rev table: db: exec: %w", err)
	}
	if _, err := d.db.Exec(createStateLatestTableSQL); err != nil {
		return fmt.Errorf("create flowstate_state_latest table: db: exec: %w", err)
	}
	if _, err := d.db.Exec(createStateLogTableSQL); err != nil {
		return fmt.Errorf("create flowstate_state_log table: db: exec: %w", err)
	}

	for _, doer := range d.doers {
		if err := doer.Init(e); err != nil {
			return fmt.Errorf("%T: init: %w", doer, err)
		}
	}
	return nil
}

func (d *Driver) Shutdown(_ context.Context) error {
	// return d.db.Close()
	return nil
}

func (d *Driver) doGetFlow(cmd *flowstate.GetFlowCommand) error {
	if cmd.StateCtx.Current.Transition.ToID == "" {
		return fmt.Errorf("transition flow to is empty")
	}

	f, err := d.Flow(cmd.StateCtx.Current.Transition.ToID)
	if err != nil {
		return err
	}

	cmd.Flow = f

	return nil
}
