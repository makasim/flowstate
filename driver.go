package flowstate

import "fmt"

type Driver interface {
	// Init must be called by NewEngine only.
	Init(e Engine) error

	GetStateByID(cmd *GetStateByIDCommand) error
	GetStateByLabels(cmd *GetStateByLabelsCommand) error
	GetStates(cmd *GetStatesCommand) error
	GetDelayedStates(cmd *GetDelayedStatesCommand) error
	Delay(cmd *DelayCommand) error
	Commit(cmd *CommitCommand) error
	GetData(cmd *GetDataCommand) error
	StoreData(cmd *AttachDataCommand) error
}

func DoCommitSubCommand(d Driver, subCmd0 Command) error {
	switch subCmd := subCmd0.(type) {
	case *NoopCommand:
		return nil
	case *CommitStateCtxCommand:
		return nil
	case *TransitCommand:
		return subCmd.Do()
	case *PauseCommand:
		return subCmd.Do()
	case *ResumeCommand:
		return subCmd.Do()
	case *ParkCommand:
		return subCmd.Do()
	case *StackCommand:
		return subCmd.Do()
	case *UnstackCommand:
		return subCmd.Do()
	case *GetDataCommand:
		if err := subCmd.Prepare(); err != nil {
			return err
		}
		return d.GetData(subCmd)
	case *AttachDataCommand:
		if err := subCmd.Prepare(); err != nil {
			return err
		}

		if subCmd.Store {
			if err := d.StoreData(subCmd); err != nil {
				return fmt.Errorf("store data: %w", err)
			}
		}

		subCmd.Do()
		return nil
	case *GetStateByIDCommand:
		if err := subCmd.Prepare(); err != nil {
			return err
		}
		return d.GetStateByID(subCmd)
	case *GetStateByLabelsCommand:
		return d.GetStateByLabels(subCmd)
	case *DelayCommand:
		if err := subCmd.Prepare(); err != nil {
			return err
		}
		return d.Delay(subCmd)
	case *ExecuteCommand:
		return fmt.Errorf("execute command not allowed inside commit")
	case *CommitCommand:
		return fmt.Errorf("commit command not allowed inside another commit")
	case *GetStatesCommand:
		return fmt.Errorf("get states command not allowed inside commit")
	case *GetDelayedStatesCommand:
		return fmt.Errorf("get delayed states command not allowed inside commit")
	default:
		return fmt.Errorf("command %T not supported", subCmd0)
	}
}
