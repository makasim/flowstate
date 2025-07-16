package flowstate

import (
	"fmt"
	"strconv"
	"strings"
)

type DataID string

type Data struct {
	noCopy

	ID     DataID
	Rev    int64
	Binary bool

	B []byte
}

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

func (cmd *ReferenceDataCommand) Do() error {
	if cmd.Data.ID == "" {
		return fmt.Errorf("data ID is empty")
	}
	if cmd.Data.Rev < 0 {
		return fmt.Errorf("data revision is negative")
	}

	cmd.StateCtx.Current.SetAnnotation(cmd.Annotation, fmt.Sprintf("data:%s:%d", cmd.Data.ID, cmd.Data.Rev))
	return nil
}

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

func (cmd *DereferenceDataCommand) Do() error {
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

func (d *Data) CopyTo(to *Data) *Data {
	to.ID = d.ID
	to.Rev = d.Rev
	to.Binary = d.Binary
	to.B = append(to.B[:0], d.B...)

	return to
}

func StoreData(d *Data) *StoreDataCommand {
	return &StoreDataCommand{
		Data: d,
	}
}

type StoreDataCommand struct {
	command

	Data *Data
}

func (cmd *StoreDataCommand) Prepare() error {
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

func GetData(d *Data) *GetDataCommand {
	return &GetDataCommand{
		Data: d,
	}
}

type GetDataCommand struct {
	command

	Data *Data
}

func (cmd *GetDataCommand) Prepare() error {
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
