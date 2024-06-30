package stddoer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/makasim/flowstate"
)

func NewDeserializer() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*flowstate.DeserializeCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

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
	})
}
