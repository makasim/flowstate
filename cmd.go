package flowstate

type Command interface {
	setDoID(doID int64)
	doID() int64
	cmd()
}

type command struct {
	propDoID int64
}

func (_ *command) cmd() {}

func (cmd *command) setDoID(doID int64) {
	cmd.propDoID = doID
}
func (cmd *command) doID() int64 {
	return cmd.propDoID
}
