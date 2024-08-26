package flowstate

func Noop(stateCtx *StateCtx) *NoopCommand {
	return &NoopCommand{
		StateCtx: stateCtx,
	}
}

type NoopCommand struct {
	command
	StateCtx *StateCtx
}

var DefaultNoopDoer DoerFunc = func(cmd0 Command) error {
	if _, ok := cmd0.(*NoopCommand); !ok {
		return ErrCommandNotSupported
	}

	return nil
}
