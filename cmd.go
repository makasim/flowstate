package flowstate

type Command interface {
	cmd()
}

type command struct {
}

func (_ *command) cmd() {}
