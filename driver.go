package flowstate

type Driver interface {
	Do(cmds ...Command) error
}
