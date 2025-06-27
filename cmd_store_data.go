package flowstate

import "fmt"

func StoreData(d *Data) *StoreDataCommand {
	return &StoreDataCommand{
		Data: d,
	}
}

type StoreDataCommand struct {
	command

	Data *Data
}

func (cmd *StoreDataCommand) prepare() error {
	if cmd.Data == nil {
		return fmt.Errorf("data is nil")
	}
	if cmd.Data.ID == "" {
		return fmt.Errorf("data ID is empty")
	}
	if len(cmd.Data.B) == 0 {
		return fmt.Errorf("data body is empty")
	}

	return nil
}
