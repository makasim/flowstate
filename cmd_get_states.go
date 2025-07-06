package flowstate

import (
	"time"
)

type GetStatesResult struct {
	States []State
	More   bool
}

func GetStatesByLabels(labels map[string]string) *GetStatesCommand {
	return (&GetStatesCommand{}).WithORLabels(labels)
}

type GetStatesCommand struct {
	command

	SinceRev   int64
	SinceTime  time.Time
	Labels     []map[string]string
	LatestOnly bool
	Limit      int

	Result *GetStatesResult
}

func (cmd *GetStatesCommand) MustResult() *GetStatesResult {
	if cmd.Result == nil {
		panic("FATAL: MustResult must be called after successful execution of the command; have you checked for errors?")
	}

	return cmd.Result
}

func (cmd *GetStatesCommand) WithSinceRev(rev int64) *GetStatesCommand {
	cmd.SinceRev = rev
	return cmd
}

func (cmd *GetStatesCommand) WithLatestOnly() *GetStatesCommand {
	cmd.LatestOnly = true
	return cmd
}

func (cmd *GetStatesCommand) WithSinceLatest() *GetStatesCommand {
	cmd.SinceRev = -1
	return cmd
}

func (cmd *GetStatesCommand) WithSinceTime(since time.Time) *GetStatesCommand {
	cmd.SinceTime = since
	return cmd
}

func (cmd *GetStatesCommand) WithORLabels(labels map[string]string) *GetStatesCommand {
	if len(labels) == 0 {
		return cmd
	}

	cmd.Labels = append(cmd.Labels, labels)
	return cmd
}

func (cmd *GetStatesCommand) WithLimit(limit int) *GetStatesCommand {
	cmd.Limit = limit
	return cmd
}

//func (cmd *GetStatesCommand) prepare() {
//	if cmd.Limit == 0 {
//		cmd.Limit = GetManyDefaultLimit
//	}
//}
