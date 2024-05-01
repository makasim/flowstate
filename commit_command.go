package flowstate

import "fmt"

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommitCommand struct {
	Commands []Command
}

func (cmd *CommitCommand) Prepare() error {
	if len(cmd.Commands) == 0 {
		return fmt.Errorf("no commands to commit")
	}

	for _, c := range cmd.Commands {
		if _, ok := c.(*CommitCommand); ok {
			return fmt.Errorf("commit command in commit command not allowed")
		}

		if err := c.Prepare(); err != nil {
			return err
		}
	}

	return nil
}
