package pgdriver

import (
	"context"
	"fmt"

	"github.com/makasim/flowstate"
)

type dataerQueries interface {
	InsertData(ctx context.Context, tx conntx, data *flowstate.Data) error
	GetData(ctx context.Context, tx conntx, id flowstate.DataID, rev int64, d *flowstate.Data) error
}

type Dataer struct {
	conn conn
	q    dataerQueries
}

func NewDataer(conn conn, q dataerQueries) *Dataer {
	return &Dataer{
		conn: conn,
		q:    q,
	}
}

func (d *Dataer) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Dataer) Shutdown(_ context.Context) error {
	return nil
}

func (d *Dataer) Do(cmd0 flowstate.Command) error {
	if cmd, ok := cmd0.(*flowstate.StoreDataCommand); ok {
		if err := cmd.Prepare(); err != nil {
			return err
		}

		if err := d.q.InsertData(context.Background(), d.conn, cmd.Data); err != nil {
			return fmt.Errorf("insert data queue: %w", err)
		}

		return nil
	}
	if cmd, ok := cmd0.(*flowstate.GetDataCommand); ok {
		if err := cmd.Prepare(); err != nil {
			return err
		}

		if err := d.q.GetData(context.Background(), d.conn, cmd.Data.ID, cmd.Data.Rev, cmd.Data); err != nil {
			return fmt.Errorf("get data query: %w", err)
		}

		return nil
	}

	return flowstate.ErrCommandNotSupported
}
