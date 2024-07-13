package flowstate

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
