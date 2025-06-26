package flowstate

import "fmt"

func GetData(d *Data) *GetDataCommand {
	return &GetDataCommand{
		Data: d,
	}
}

type GetDataCommand struct {
	command

	Data *Data
}

func (cmd *GetDataCommand) prepare() error {
	if cmd.Data == nil {
		return fmt.Errorf("data is nil")
	}
	if cmd.Data.ID == "" {
		return fmt.Errorf("data ID is empty")
	}
	if cmd.Data.Rev < 0 {
		return fmt.Errorf("data revision is negative")
	}

	return nil
}
