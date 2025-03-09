package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Getter{}

type getterQueries interface {
	GetStateByID(ctx context.Context, tx conntx, id flowstate.StateID, rev int64, s *flowstate.State) error
	GetStatesByLabels(ctx context.Context, tx conntx, orLabels []map[string]string, sinceRev int64, ss []flowstate.State) ([]flowstate.State, error)
	GetLatestStatesByLabels(ctx context.Context, tx conntx, orLabels []map[string]string, sinceRev int64, ss []flowstate.State) ([]flowstate.State, error)
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
	case *flowstate.GetCommand:
		return d.doGet(cmd)
	case *flowstate.GetManyCommand:
		return d.doGetMany(cmd)
	default:
		return flowstate.ErrCommandNotSupported
	}
}

func (d *Getter) doGet(cmd *flowstate.GetCommand) error {
	if false == (len(cmd.Labels) == 0 || cmd.ID == "") {
		return fmt.Errorf("labels or id mus be set")
	}

	s := flowstate.State{}
	if len(cmd.Labels) > 0 {
		rev := cmd.Rev
		if rev == 0 {
			rev = -1
		}

		ss := []flowstate.State{s}
		ss, err := d.q.GetStatesByLabels(context.Background(), d.conn, []map[string]string{cmd.Labels}, rev, ss)
		if err != nil {
			return fmt.Errorf("get states by labels query: %w", err)
		} else if len(ss) == 0 {
			return fmt.Errorf("state not found")
		}
		s = ss[0]
	} else {
		if err := d.q.GetStateByID(context.Background(), d.conn, cmd.ID, cmd.Rev, &s); err != nil {
			return fmt.Errorf("get state by id query: %w", err)
		}
	}

	s.CopyToCtx(cmd.StateCtx)
	return nil
}

func (d *Getter) doGetMany(cmd *flowstate.GetManyCommand) error {
	cmd.Prepare()

	if !cmd.SinceTime.IsZero() {
		return fmt.Errorf("since time is not supported")
	}

	ss := make([]flowstate.State, cmd.Limit+1)
	if cmd.LatestOnly {
		var err error
		ss, err = d.q.GetLatestStatesByLabels(context.Background(), d.conn, cmd.Labels, cmd.SinceRev, ss)
		if err != nil {
			return fmt.Errorf("get latest states by labels query: %w", err)
		}
	} else {
		var err error
		ss, err = d.q.GetStatesByLabels(context.Background(), d.conn, cmd.Labels, cmd.SinceRev, ss)
		if err != nil {
			return fmt.Errorf("get states by labels query: %w", err)
		}
	}

	var more bool
	if len(ss) > cmd.Limit {
		ss = ss[:cmd.Limit]
		more = true
	}

	cmd.SetResult(&flowstate.GetManyResult{
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
