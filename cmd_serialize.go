package flowstate

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func Serialize(serializableStateCtx, stateCtx *StateCtx, annotation string) *SerializeCommand {
	return &SerializeCommand{
		SerializableStateCtx: serializableStateCtx,
		StateCtx:             stateCtx,
		Annotation:           annotation,
	}

}

type SerializeCommand struct {
	command
	SerializableStateCtx *StateCtx
	StateCtx             *StateCtx
	Annotation           string
}

func (cmd *SerializeCommand) do() error {
	if cmd.Annotation == `` {
		return fmt.Errorf("store annotation name empty")
	}
	if cmd.StateCtx.Current.Annotations[cmd.Annotation] != `` {
		return fmt.Errorf("store annotation already set")
	}

	b, err := json.Marshal(cmd.SerializableStateCtx)
	if err != nil {
		return fmt.Errorf("json marshal prev state ctx: %s", err)
	}
	serialized := base64.StdEncoding.EncodeToString(b)

	cmd.StateCtx.Current.SetAnnotation(cmd.Annotation, serialized)

	return nil
}
