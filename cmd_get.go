package flowstate

func GetByID(stateCtx *StateCtx, id StateID, rev int64) *GetCommand {
	return (&GetCommand{
		StateCtx: stateCtx,
	}).WithID(id).WithRev(rev)
}

func GetByLabels(stateCtx *StateCtx, labels map[string]string) *GetCommand {
	return (&GetCommand{
		StateCtx: stateCtx,
	}).WithLabels(labels)
}

type GetCommand struct {
	command

	ID     StateID
	Rev    int64
	Labels map[string]string

	StateCtx *StateCtx
}

func (c *GetCommand) WithID(id StateID) *GetCommand {
	c.Labels = nil

	c.ID = id
	return c
}

func (c *GetCommand) WithRev(rev int64) *GetCommand {
	c.Rev = rev
	return c
}

func (c *GetCommand) WithLabels(labels map[string]string) *GetCommand {
	c.ID = ``
	c.Rev = 0

	c.Labels = labels
	return c
}
