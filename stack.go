package flowstate

import (
	"encoding/base64"
	"fmt"
)

func Stack(carrierStateCtx, stackStateCtx *StateCtx, annotation string) *StackCommand {
	return &StackCommand{
		StackedStateCtx: stackStateCtx,
		CarrierStateCtx: carrierStateCtx,
		Annotation:      annotation,
	}

}

type StackCommand struct {
	command
	StackedStateCtx *StateCtx
	CarrierStateCtx *StateCtx
	Annotation      string
}

func (cmd *StackCommand) Do() error {
	if cmd.Annotation == `` {
		return fmt.Errorf("stack annotation name empty")
	}
	if cmd.CarrierStateCtx.Current.Annotations[cmd.Annotation] != `` {
		return fmt.Errorf("stack annotation '%s' already set", cmd.Annotation)
	}

	b := MarshalStateCtx(cmd.StackedStateCtx, nil)
	b64 := base64.StdEncoding.EncodeToString(b)
	cmd.CarrierStateCtx.Current.SetAnnotation(cmd.Annotation, b64)

	return nil
}

func Unstack(carrierStateCtx, unstackStateCtx *StateCtx, annotation string) *UnstackCommand {
	return &UnstackCommand{
		CarrierStateCtx: carrierStateCtx,
		UnstackStateCtx: unstackStateCtx,
		Annotation:      annotation,
	}

}

type UnstackCommand struct {
	command
	CarrierStateCtx *StateCtx
	UnstackStateCtx *StateCtx
	Annotation      string
}

func (cmd *UnstackCommand) Do() error {
	b64 := cmd.CarrierStateCtx.Current.Annotations[cmd.Annotation]
	if b64 == `` {
		return fmt.Errorf("stack annotation value empty")
	}

	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("cannot base64 decode state in annotation '%s': %s", cmd.Annotation, err)
	}

	if err := UnmarshalStateCtx(b, cmd.UnstackStateCtx); err != nil {
		return fmt.Errorf("cannot unmarshal stack in annotation '%s': %s", cmd.Annotation, err)
	}

	cmd.CarrierStateCtx.Current.Annotations[cmd.Annotation] = ``

	return nil
}
