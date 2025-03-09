package flowstate

import (
	"fmt"
	"time"
)

const GetManyDefaultLimit = 50

type GetManyResult struct {
	States []State
	More   bool
}

func GetManyByLabels(labels map[string]string) *GetManyCommand {
	return (&GetManyCommand{}).WithORLabels(labels)
}

type GetManyCommand struct {
	command

	SinceRev   int64
	SinceTime  time.Time
	Labels     []map[string]string
	LatestOnly bool
	Limit      int

	result *GetManyResult
}

func (cmd *GetManyCommand) Result() (*GetManyResult, error) {
	if cmd.result == nil {
		return nil, fmt.Errorf("no result set")
	}

	return cmd.result, nil
}

func (cmd *GetManyCommand) SetResult(result *GetManyResult) {
	cmd.result = result
}

func (cmd *GetManyCommand) WithSinceRev(rev int64) *GetManyCommand {
	cmd.SinceRev = rev
	return cmd
}

func (cmd *GetManyCommand) WithSinceLatest() *GetManyCommand {
	cmd.SinceRev = -1
	return cmd
}

func (cmd *GetManyCommand) WithSinceTime(since time.Time) *GetManyCommand {
	cmd.SinceTime = since
	return cmd
}

func (cmd *GetManyCommand) WithORLabels(labels map[string]string) *GetManyCommand {
	if len(labels) == 0 {
		return cmd
	}

	cmd.Labels = append(cmd.Labels, labels)
	return cmd
}

func (cmd *GetManyCommand) Prepare() {
	if cmd.Limit == 0 {
		cmd.Limit = GetManyDefaultLimit
	}
}
