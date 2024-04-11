package flowstate

type BehaviorID string

type Behavior interface {
	Execute(taskCtx *TaskCtx) (Command, error)
}

type BehaviorFunc func(taskCtx *TaskCtx) (Command, error)

func (f BehaviorFunc) Execute(taskCtx *TaskCtx) (Command, error) {
	return f(taskCtx)
}
