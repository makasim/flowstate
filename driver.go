package flowstate

import (
	"fmt"
)

type Driver interface {
	// Init must be called by NewEngine only.
	Init(e *Engine) error

	GetStateByID(cmd *GetStateByIDCommand) error
	GetStateByLabels(cmd *GetStateByLabelsCommand) error
	GetStates(cmd *GetStatesCommand) error
	GetDelayedStates(cmd *GetDelayedStatesCommand) error
	Delay(cmd *DelayCommand) error
	Commit(cmd *CommitCommand) error
	GetData(cmd *GetDataCommand) error
	StoreData(cmd *StoreDataCommand) error
}

func DoCommitSubCommand(d Driver, cmd0 Command) error {
	switch cmd := cmd0.(type) {
	case *NoopCommand:
		return nil
	case *TransitCommand:
		return cmd.Do()
	case *ParkCommand:
		return cmd.Do()
	case *StackCommand:
		return cmd.Do()
	case *UnstackCommand:
		return cmd.Do()
	case *StoreDataCommand:
		if store, err := cmd.Prepare(); err != nil {
			return err
		} else if !store {
			return nil
		}
		if err := d.StoreData(cmd); err != nil {
			return err
		}
		cmd.post()
		return nil
	case *GetDataCommand:
		if get, err := cmd.Prepare(); err != nil {
			return err
		} else if !get {
			return nil
		}
		return d.GetData(cmd)
	case *GetStateByIDCommand:
		if err := cmd.Prepare(); err != nil {
			return err
		}
		return d.GetStateByID(cmd)
	case *GetStateByLabelsCommand:
		return d.GetStateByLabels(cmd)
	case *DelayCommand:
		if err := cmd.Prepare(); err != nil {
			return err
		}
		return d.Delay(cmd)
	case *ExecuteCommand:
		return fmt.Errorf("execute command not allowed inside commit")
	case *CommitCommand:
		return fmt.Errorf("commit command not allowed inside another commit")
	case *GetStatesCommand:
		return fmt.Errorf("get states command not allowed inside commit")
	case *GetDelayedStatesCommand:
		return fmt.Errorf("get delayed states command not allowed inside commit")
	default:
		return fmt.Errorf("command %T not supported", cmd0)
	}
}
