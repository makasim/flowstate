package flowstate

type BehaviorID string

type Behavior interface {
	Execute(taskCtx *TaskCtx) error
}

type BehaviorFunc func(taskCtx *TaskCtx) error

func (f BehaviorFunc) Execute(taskCtx *TaskCtx) error {
	return f(taskCtx)
}
