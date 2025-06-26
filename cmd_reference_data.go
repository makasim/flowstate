package flowstate

import "fmt"

func ReferenceData(stateCtx *StateCtx, data *Data, annotation string) *ReferenceDataCommand {
	return &ReferenceDataCommand{
		StateCtx:   stateCtx,
		Data:       data,
		Annotation: annotation,
	}

}

type ReferenceDataCommand struct {
	command
	StateCtx   *StateCtx
	Data       *Data
	Annotation string
}

func (cmd *ReferenceDataCommand) do() error {
	if cmd.Data.ID == "" {
		return fmt.Errorf("data ID is empty")
	}
	if cmd.Data.Rev < 0 {
		return fmt.Errorf("data revision is negative")
	}

	cmd.StateCtx.Current.SetAnnotation(cmd.Annotation, fmt.Sprintf("data:%s:%d", cmd.Data.ID, cmd.Data.Rev))
	return nil
}
