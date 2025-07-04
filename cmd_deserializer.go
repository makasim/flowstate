package flowstate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Deserialize(stateCtx, deserializedStateCtx *StateCtx, annotation string) *DeserializeCommand {
	return &DeserializeCommand{
		StateCtx:             stateCtx,
		DeserializedStateCtx: deserializedStateCtx,
		Annotation:           annotation,
	}

}

type DeserializeCommand struct {
	command
	StateCtx             *StateCtx
	DeserializedStateCtx *StateCtx
	Annotation           string
}

func (cmd *DeserializeCommand) do() error {
	serializedState := cmd.StateCtx.Current.Annotations[cmd.Annotation]
	if serializedState == `` {
		return fmt.Errorf("store annotation value empty")
	}

	b, err := base64.StdEncoding.DecodeString(serializedState)
	if err != nil {
		return fmt.Errorf("base64 decode: %s", err)
	}

	if err := json.Unmarshal(b, cmd.DeserializedStateCtx); err != nil {
		return fmt.Errorf("json unmarshal: %s", err)
	}

	cmd.StateCtx.Current.Annotations[cmd.Annotation] = ``

	return nil
}
