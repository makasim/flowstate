package flowstate

type FlowID string

type Flow interface {
	Execute(stateCtx *StateCtx, e *Engine) (Command, error)
}

type FlowFunc func(stateCtx *StateCtx, e *Engine) (Command, error)

func (f FlowFunc) Execute(stateCtx *StateCtx, e *Engine) (Command, error) {
	return f(stateCtx, e)
}
