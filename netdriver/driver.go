package netdriver

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

func New(httpHost string) *Driver {
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

func (d *Driver) GetStates(cmd *flowstate.GetStatesCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/GetStates")
}

func (d *Driver) GetDelayedStates(cmd *flowstate.GetDelayedStatesCommand) error {
	return d.do(cmd, "/flowstate.v1.Driver/GetDelayedStates")
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

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if http.StatusOK != resp.StatusCode {
		code, message, err := unmarshalError(b)
		if err != nil {
			return fmt.Errorf("response status code: %d; unmarshal error: %s", resp.StatusCode, err)
		}

		switch {
		case code == "not_found" && strings.Contains(message, flowstate.ErrNotFound.Error()):
			return flowstate.ErrNotFound
		case code == "aborted" && strings.HasPrefix(message, "rev mismatch: "):
			_, idsStr, _ := strings.Cut(message, "rev mismatch: ")
			splitIds := strings.Split(idsStr, ",")
			ids := make([]flowstate.StateID, 0, len(splitIds))
			for i := range splitIds {
				id := strings.TrimSpace(splitIds[i])
				if id == "" {
					continue
				}

				ids = append(ids, flowstate.StateID(id))
			}

			return flowstate.ErrRevMismatch{IDS: ids}
		}

		return fmt.Errorf("%s: %s", code, message)
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
	case *flowstate.TransitCommand:
		resCmd, ok := resCmd0.(*flowstate.TransitCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a TransitCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.PauseCommand:
		resCmd, ok := resCmd0.(*flowstate.PauseCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a PauseCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.ResumeCommand:
		resCmd, ok := resCmd0.(*flowstate.ResumeCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a ResumeCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.EndCommand:
		resCmd, ok := resCmd0.(*flowstate.EndCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a EndCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.ExecuteCommand:
		resCmd, ok := resCmd0.(*flowstate.ExecuteCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a ExecuteCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.DelayCommand:
		resCmd, ok := resCmd0.(*flowstate.DelayCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a DelayCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		resCmd.DelayingState.CopyTo(&inCmd.DelayingState)
		return nil
	case *flowstate.CommitCommand:
		resCmd, ok := resCmd0.(*flowstate.CommitCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a CommitCommand")
		}
		if len(resCmd.Commands) != len(inCmd.Commands) {
			return fmt.Errorf("resCmd.Commands length mismatch: got %d, want %d", len(resCmd.Commands), len(inCmd.Commands))
		}

		for i := range inCmd.Commands {
			if err := syncResult(inCmd.Commands[i], resCmd.Commands[i]); err != nil {
				return fmt.Errorf("%d# command %T: %w", i, inCmd.Commands[i], err)
			}
		}

		return nil
	case *flowstate.NoopCommand:
		resCmd, ok := resCmd0.(*flowstate.ExecuteCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a ExecuteCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.StackCommand:
		resCmd, ok := resCmd0.(*flowstate.StackCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a StackCommand")
		}

		resCmd.CarrierStateCtx.CopyTo(inCmd.CarrierStateCtx)
		resCmd.StackedStateCtx.CopyTo(inCmd.StackedStateCtx)
		return nil
	case *flowstate.UnstackCommand:
		resCmd, ok := resCmd0.(*flowstate.UnstackCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a UnstackCommand")
		}

		resCmd.CarrierStateCtx.CopyTo(inCmd.CarrierStateCtx)
		resCmd.UnstackStateCtx.CopyTo(inCmd.UnstackStateCtx)
		return nil
	case *flowstate.GetDataCommand:
		resCmd, ok := resCmd0.(*flowstate.GetDataCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a GetDataCommand")
		}

		resCmd.Data.CopyTo(inCmd.Data)
		return nil
	case *flowstate.StoreDataCommand:
		resCmd, ok := resCmd0.(*flowstate.StoreDataCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a StoreDataCommand")
		}

		resCmd.Data.CopyTo(inCmd.Data)
		return nil
	case *flowstate.ReferenceDataCommand:
		resCmd, ok := resCmd0.(*flowstate.ReferenceDataCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a ReferenceDataCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		resCmd.Data.CopyTo(inCmd.Data)
		return nil
	case *flowstate.DereferenceDataCommand:
		resCmd, ok := resCmd0.(*flowstate.DereferenceDataCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a DereferenceDataCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		resCmd.Data.CopyTo(inCmd.Data)
		return nil
	case *flowstate.GetStateByIDCommand:
		resCmd, ok := resCmd0.(*flowstate.GetStateByIDCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a GetStateByIDCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.GetStateByLabelsCommand:
		resCmd, ok := resCmd0.(*flowstate.GetStateByLabelsCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a GetStateByLabelsCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	case *flowstate.GetStatesCommand:
		resCmd, ok := resCmd0.(*flowstate.GetStatesCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a GetStatesCommand")
		}

		inCmd.Result = resCmd.Result
		return nil
	case *flowstate.GetDelayedStatesCommand:
		resCmd, ok := resCmd0.(*flowstate.GetDelayedStatesCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a GetDelayedStatesCommand")
		}

		inCmd.Result = resCmd.Result
		return nil
	case *flowstate.CommitStateCtxCommand:
		resCmd, ok := resCmd0.(*flowstate.CommitStateCtxCommand)
		if !ok {
			return fmt.Errorf("resCmd is not a CommitStateCtxCommand")
		}

		resCmd.StateCtx.CopyTo(inCmd.StateCtx)
		return nil
	default:
		return fmt.Errorf("unknown inCmd: %T", inCmd0)
	}
}
