package pgdriver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Getter{}

type getterQueries interface {
	GetStateByID(ctx context.Context, tx conntx, id flowstate.StateID, rev int64, s *flowstate.State) error
	GetStatesByLabels(ctx context.Context, tx conntx, orLabels []map[string]string, sinceRev int64, sinceTime time.Time, ss []flowstate.State) ([]flowstate.State, error)
	GetLatestStatesByLabels(ctx context.Context, tx conntx, orLabels []map[string]string, sinceRev int64, sinceTime time.Time, ss []flowstate.State) ([]flowstate.State, error)
}

type Getter struct {
	conn conn
	q    getterQueries
}

func NewGetter(conn conn, q getterQueries) *Getter {
	return &Getter{
		conn: conn,
		q:    q,
	}
}

func (d *Getter) Do(cmd0 flowstate.Command) error {
	switch cmd := cmd0.(type) {
	case *flowstate.GetStateByIDCommand:
		return d.doGetStateByID(cmd)
	case *flowstate.GetStateByLabelsCommand:
		return d.doGetStateByLabels(cmd)
	case *flowstate.GetStatesCommand:
		return d.doGetMany(cmd)
	default:
		return flowstate.ErrCommandNotSupported
	}
}

func (d *Getter) doGetStateByID(cmd *flowstate.GetStateByIDCommand) error {
	if err := cmd.Prepare(); err != nil {
		return fmt.Errorf("get state by id: preapare: %w", err)
	}

	s := flowstate.State{}
	if err := d.q.GetStateByID(context.Background(), d.conn, cmd.ID, cmd.Rev, &s); errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w; id=%v rev=%d", flowstate.ErrNotFound, cmd.ID, cmd.Rev)
	} else if err != nil {
		return fmt.Errorf("get state by id: query: %w", err)
	}

	s.CopyToCtx(cmd.StateCtx)
	return nil
}

func (d *Getter) doGetStateByLabels(cmd *flowstate.GetStateByLabelsCommand) error {
	s := flowstate.State{}

	ss := []flowstate.State{s}
	ss, err := d.q.GetStatesByLabels(context.Background(), d.conn, []map[string]string{cmd.Labels}, -1, time.Time{}, ss)
	if err != nil {
		return fmt.Errorf("get states by labels query: %w", err)
	} else if len(ss) == 0 {
		return fmt.Errorf("%w; labels=%v", flowstate.ErrNotFound, cmd.Labels)
	}
	s = ss[0]

	s.CopyToCtx(cmd.StateCtx)
	return nil
}

func (d *Getter) doGetMany(cmd *flowstate.GetStatesCommand) error {
	cmd.Prepare()

	ss := make([]flowstate.State, cmd.Limit+1)
	if cmd.LatestOnly {
		var err error
		ss, err = d.q.GetLatestStatesByLabels(context.Background(), d.conn, cmd.Labels, cmd.SinceRev, cmd.SinceTime, ss)
		if err != nil {
			return fmt.Errorf("get latest states by labels query: %w", err)
		}
	} else {
		var err error
		ss, err = d.q.GetStatesByLabels(context.Background(), d.conn, cmd.Labels, cmd.SinceRev, cmd.SinceTime, ss)
		if err != nil {
			return fmt.Errorf("get states by labels query: %w", err)
		}
	}

	var more bool
	if len(ss) > cmd.Limit {
		ss = ss[:cmd.Limit]
		more = true
	}

	cmd.SetResult(&flowstate.GetStatesResult{
		States: ss,
		More:   more,
	})

	return nil
}

func (d *Getter) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Getter) Shutdown(_ context.Context) error {
	return nil
}
