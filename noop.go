package flowstate

import (
	"reflect"
)

func Noop(stateCtx *StateCtx) *NoopCommand {
	return &NoopCommand{
		StateCtx: stateCtx,
	}
}

type NoopCommand struct {
	StateCtx *StateCtx
}

func (cmd *NoopCommand) Prepare() error {
	return nil
}

type Nooper struct {
	t reflect.Type
}

func NewNooper() *Nooper {
	return &Nooper{
		t: reflect.TypeOf(&NoopCommand{}),
	}
}

func (d *Nooper) Do(cmd0 Command) (*StateCtx, error) {
	if d.t.String() != reflect.TypeOf(cmd0).String() {
		return nil, ErrCommandNotSupported
	}

	return nil, nil
}
