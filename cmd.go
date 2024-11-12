package flowstate

type Command interface {
	setSessID(id int64)
	SessID() int64
	cmd()
}

type command struct {
	sessID int64
}

func (_ *command) cmd() {}

func (cmd *command) setSessID(doID int64) {
	cmd.sessID = doID
}

func (cmd *command) SessID() int64 {
	return cmd.sessID
}
