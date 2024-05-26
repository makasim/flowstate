package flowstate

import (
	"fmt"
)

type Engine struct {
	d Doer
}

func NewEngine(d Doer) *Engine {
	return &Engine{
		d: d,
	}
}

func (e *Engine) Execute(stateCtx *StateCtx) error {
	_, err := e.do(Execute(stateCtx))
	return err
}

func (e *Engine) Do(cmds ...Command) error {
	if len(cmds) == 0 {
		return fmt.Errorf("no commands to do")
	}

	for _, cmd := range cmds {
		if _, err := e.do(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) Watch(rev int64, labels map[string]string) (WatchListener, error) {
	cmd := Watch(rev, labels)
	if err := e.Do(cmd); err != nil {
		return nil, err
	}

	return cmd.Listener, nil
}

func (e *Engine) do(cmd0 Command) (*StateCtx, error) {
	if cmd1, ok := cmd0.(*CommitCommand); ok {
		if _, err := e.d.Do(cmd1); err != nil {
			return nil, err
		}

		var returnStateCtx *StateCtx
		for _, stateCtx := range cmd1.NextStateCtxs {
			if returnStateCtx == nil {
				returnStateCtx = stateCtx
			} else if stateCtx != nil {
				if _, err := e.do(Execute(stateCtx)); err != nil {
					return nil, nil
				}
			}
		}

		return returnStateCtx, nil
	}

	return e.d.Do(cmd0)
}
