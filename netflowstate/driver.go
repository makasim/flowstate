package netflowstate

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/makasim/flowstate"
)

var _ flowstate.Driver = (*Driver)(nil)

type Driver struct {
	flowstate.FlowRegistry

	httpHost string

	c *http.Client
}

func NewDriver(httpHost string) *Driver {
	return &Driver{
		httpHost: httpHost,

		c: &http.Client{},
	}
}

func (d *Driver) Init(_ flowstate.Engine) error {
	return nil
}

func (d *Driver) GetStateByID(cmd *flowstate.GetStateByIDCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/GetStateByID")
}

func (d *Driver) GetStateByLabels(cmd *flowstate.GetStateByLabelsCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/GetStateByLabels")
}

func (d *Driver) GetStates(cmd *flowstate.GetStatesCommand) (*flowstate.GetStatesResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *Driver) GetDelayedStates(cmd *flowstate.GetDelayedStatesCommand) (*flowstate.GetDelayedStatesResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *Driver) Delay(cmd *flowstate.DelayCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/Delay")
}

func (d *Driver) Commit(cmd *flowstate.CommitCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/Commit")
}

func (d *Driver) GetData(cmd *flowstate.GetDataCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/GetData")
}

func (d *Driver) StoreData(cmd *flowstate.StoreDataCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/StoreData")
}

func (d *Driver) do(cmd flowstate.Command, path string) error {
	b := flowstate.MarshalCommand(cmd, nil)
	req, err := http.NewRequest(`POST`, strings.TrimRight(d.httpHost, `/`)+path, bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}

	resp, err := d.c.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if http.StatusOK != resp.StatusCode {
		// TODO: handle error
		return fmt.Errorf("got status code: %d", resp.StatusCode)
	}

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	resCmd, err := flowstate.UnmarshalCommand(b)
	if err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	if err := syncResult(cmd, resCmd); err != nil {
		return fmt.Errorf("sync result: %w", err)
	}

	return nil
}

func syncResult(inCmd0, resCmd0 flowstate.Command) error {
	switch inCmd := inCmd0.(type) {
	case *flowstate.GetStateByIDCommand:
		resCmd, ok := resCmd0.(*flowstate.GetStateByIDCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a GetStateByIDCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	default:
		return fmt.Errorf("unknown inCmd: %T", inCmd0)
	}
}
