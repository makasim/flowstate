package flowstate

type Driver interface {
	Commit(cmds ...Command) error
}
