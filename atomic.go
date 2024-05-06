package flowstate

import (
	"fmt"
	"strconv"
)

func AddAtomicInt64(name string, val int64, taskCtx *TaskCtx) *AddAtomicInt64Command {
	return &AddAtomicInt64Command{
		TaskCtx: taskCtx,
		Name:    name,
		Val:     val,
	}

}

type AddAtomicInt64Command struct {
	TaskCtx *TaskCtx
	Name    string
	Val     int64
}

func (cmd *AddAtomicInt64Command) Prepare() error {
	return nil
}

func GetAtomicInt64(name string, taskCtx *TaskCtx) (int64, error) {
	val, ok := taskCtx.Current.Annotations[`atomic.`+name]
	if !ok {
		return 0, fmt.Errorf("atomic value %q not found", name)
	}

	val1, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("atomic value %s=%s is not an integer", name, val)
	}

	return val1, nil
}

func SetAtomicInt64(name string, val int64, taskCtx *TaskCtx) {
	taskCtx.Current.SetAnnotation(
		`atomic.`+name,
		strconv.FormatInt(val, 10),
	)
}
