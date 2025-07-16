package flowstate

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/oklog/ulid/v2"
)

type DataID string

type Data struct {
	noCopy

	ID     DataID
	Rev    int64
	Binary bool

	B []byte
}

func (d *Data) CopyTo(to *Data) *Data {
	to.ID = d.ID
	to.Rev = d.Rev
	to.Binary = d.Binary
	to.B = append(to.B[:0], d.B...)

	return to
}

func AttachData(stateCtx *StateCtx, data *Data, alias string) *AttachDataCommand {
	return &AttachDataCommand{
		StateCtx: stateCtx,
		Data:     data,
		Alias:    alias,

		Store: true,
	}
}

type AttachDataCommand struct {
	command
	StateCtx *StateCtx
	Data     *Data
	Alias    string
	Store    bool
}

func (cmd *AttachDataCommand) WithoutStore() *AttachDataCommand {
	cmd.Store = false
	return cmd
}

func (cmd *AttachDataCommand) Prepare() error {
	if cmd.Alias == "" {
		return fmt.Errorf("alias is empty")
	}
	if cmd.Data.ID == "" {
		cmd.Data.ID = DataID(ulid.Make().String())
	}
	if cmd.Data.Rev < 0 {
		return fmt.Errorf("Data.Rev is negative")
	}
	if cmd.Data.B == nil || len(cmd.Data.B) == 0 {
		return fmt.Errorf("Data.B is empty")
	}

	return nil
}

func (cmd *AttachDataCommand) Do() {
	cmd.StateCtx.Current.SetAnnotation(
		dataAnnotation(cmd.Alias),
		string(cmd.Data.ID)+":"+strconv.FormatInt(cmd.Data.Rev, 10),
	)
}

func GetData(stateCtx *StateCtx, data *Data, alias string) *GetDataCommand {
	return &GetDataCommand{
		StateCtx: stateCtx,
		Data:     data,
		Alias:    alias,
	}

}

type GetDataCommand struct {
	command
	StateCtx *StateCtx
	Data     *Data
	Alias    string
}

func (cmd *GetDataCommand) Prepare() error {
	if cmd.Data == nil {
		return fmt.Errorf("data is nil")
	}
	if cmd.Alias == "" {
		return fmt.Errorf("alias is empty")
	}

	annotKey := dataAnnotation(cmd.Alias)
	idRevStr := cmd.StateCtx.Current.Annotations[annotKey]
	if idRevStr == "" {
		return fmt.Errorf("annotation %q is not set", annotKey)
	}

	sepIdx := strings.LastIndexAny(idRevStr, ":")
	log.Println(sepIdx, sepIdx < 1, sepIdx+1 == len(idRevStr))
	if sepIdx < 1 || sepIdx+1 == len(idRevStr) {
		return fmt.Errorf("annotation %q contains invalid data reference; got %q", annotKey, idRevStr)
	}

	id := DataID(idRevStr[:sepIdx])
	rev, err := strconv.ParseInt(idRevStr[sepIdx+1:], 10, 64)
	if err != nil {
		return fmt.Errorf("annotation %q contains invalid data revision; got %q: %w", annotKey, idRevStr[sepIdx+1:], err)
	}

	cmd.Data.ID = id
	cmd.Data.Rev = rev

	return nil
}

func dataAnnotation(alias string) string {
	return "flowstate.data." + string(alias)
}
