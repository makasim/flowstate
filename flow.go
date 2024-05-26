package flowstate

type FlowID string

type Flow interface {
	Execute(stateCtx *StateCtx) (Command, error)
}

type FlowFunc func(stateCtx *StateCtx) (Command, error)

func (f FlowFunc) Execute(stateCtx *StateCtx) (Command, error) {
	return f(stateCtx)
}
