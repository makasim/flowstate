package memdriver

import (
	"context"
	"fmt"
	"sync"

	"github.com/makasim/flowstate"
)

type DataLog struct {
	sync.Mutex
	rev     int64
	entries []*flowstate.Data
}

func NewDataLog() *DataLog {
	return &DataLog{}
}

func (l *DataLog) Init(_ *flowstate.Engine) error {
	return nil
}

func (l *DataLog) Shutdown(_ context.Context) error {
	return nil
}

func (l *DataLog) Do(cmd0 flowstate.Command) error {
	if cmd, ok := cmd0.(*flowstate.StoreDataCommand); ok {
		return l.doStore(cmd)
	}
	if cmd, ok := cmd0.(*flowstate.GetDataCommand); ok {
		return l.doGet(cmd)
	}

	return flowstate.ErrCommandNotSupported
}

func (l *DataLog) Append(data *flowstate.Data) {
	l.Lock()
	defer l.Unlock()

	l.rev++
	data.Rev = l.rev
	l.entries = append(l.entries, data.CopyTo(&flowstate.Data{}))
}

func (l *DataLog) Get(id flowstate.DataID, rev int64) (*flowstate.Data, error) {
	l.Lock()
	defer l.Unlock()

	for _, data := range l.entries {
		if data.ID == id && data.Rev == rev {
			return data, nil
		}
	}

	return nil, fmt.Errorf("data not found")
}

func (l *DataLog) doGet(cmd *flowstate.GetDataCommand) error {
	if err := cmd.Prepare(); err != nil {
		return err
	}

	data, err := l.Get(cmd.Data.ID, cmd.Data.Rev)
	if err != nil {
		return err
	}

	data.CopyTo(cmd.Data)
	return nil
}

func (l *DataLog) doStore(cmd *flowstate.StoreDataCommand) error {
	if err := cmd.Prepare(); err != nil {
		return err
	}

	l.Append(cmd.Data)
	return nil
}
