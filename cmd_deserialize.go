package flowstate

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
