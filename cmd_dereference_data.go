package flowstate

import (
	"fmt"
	"strconv"
	"strings"
)

func DereferenceData(stateCtx *StateCtx, data *Data, annotation string) *DereferenceDataCommand {
	return &DereferenceDataCommand{
		StateCtx:   stateCtx,
		Data:       data,
		Annotation: annotation,
	}

}

type DereferenceDataCommand struct {
	command
	StateCtx   *StateCtx
	Data       *Data
	Annotation string
}

func (cmd *DereferenceDataCommand) do() error {
	serializedData := cmd.StateCtx.Current.Annotations[cmd.Annotation]
	if serializedData == "" {
		return fmt.Errorf("data is not serialized")
	}

	splits := strings.SplitN(serializedData, ":", 3)
	if len(splits) != 3 {
		return fmt.Errorf("data is not serialized correctly")
	}
	if splits[0] != "data" {
		return fmt.Errorf("data is not serialized correctly")
	}
	if splits[1] == "" {
		return fmt.Errorf("serialized data ID is empty")
	}
	if splits[2] == "" {
		return fmt.Errorf("serialized data revision is empty")
	}

	dRev, err := strconv.ParseInt(splits[2], 10, 64)
	if err != nil {
		return fmt.Errorf("serialized data revision is not integer: %w", err)
	}

	cmd.Data.ID = DataID(splits[1])
	cmd.Data.Rev = dRev

	return nil
}
