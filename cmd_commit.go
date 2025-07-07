package flowstate

import "fmt"

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommittableCommand interface {
	CommittableStateCtx() *StateCtx
}

type CommitCommand struct {
	command
	Commands []Command
}

func (cmd *CommitCommand) setSessID(id int64) {
	cmd.command.setSessID(id)
	for _, subCmd := range cmd.Commands {
		subCmd.setSessID(id)
	}
}

func CommitStateCtx(stateCtx *StateCtx) *CommitStateCtxCommand {
	return &CommitStateCtxCommand{
		StateCtx: stateCtx,
	}
}

type CommitStateCtxCommand struct {
	command
	StateCtx *StateCtx
}

func (cmd *CommitStateCtxCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
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
	case *EndCommand:
		return subCmd.Do()
	case *SerializeCommand:
		return subCmd.Do()
	case *DeserializeCommand:
		return subCmd.Do()
	case *DereferenceDataCommand:
		return subCmd.Do()
	case *ReferenceDataCommand:
		return subCmd.Do()
	case *GetDataCommand:
		if err := subCmd.Prepare(); err != nil {
			return err
		}
		return d.GetData(subCmd)
	case *StoreDataCommand:
		if err := subCmd.Prepare(); err != nil {
			return err
		}
		return d.StoreData(subCmd)
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
