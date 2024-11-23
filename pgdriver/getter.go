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
	cmd, ok := cmd0.(*flowstate.GetCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

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

func (d *Getter) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Getter) Shutdown(_ context.Context) error {
	return nil
}
