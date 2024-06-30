package stddoer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/makasim/flowstate"
)

func NewSerializer() flowstate.Doer {
	return flowstate.DoerFunc(func(cmd0 flowstate.Command) error {
		cmd, ok := cmd0.(*flowstate.SerializeCommand)
		if !ok {
			return flowstate.ErrCommandNotSupported
		}

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
	})
}
