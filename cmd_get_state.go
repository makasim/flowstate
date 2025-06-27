package flowstate

import "fmt"

func GetStateByID(stateCtx *StateCtx, id StateID, rev int64) *GetStateByIDCommand {
	return &GetStateByIDCommand{
		ID:       id,
		Rev:      rev,
		StateCtx: stateCtx,
	}
}

type GetStateByIDCommand struct {
	command

	ID  StateID
	Rev int64

	StateCtx *StateCtx
}

func (cmd *GetStateByIDCommand) prepare() error {
	if cmd.ID == "" {
		return fmt.Errorf(`id is empty`)
	}
	if cmd.Rev < 0 {
		return fmt.Errorf(`rev must be >= 0`)
	}
	return nil
}

func (cmd *GetStateByIDCommand) Result() (*StateCtx, error) {
	if cmd.StateCtx == nil {
		return nil, fmt.Errorf(`cmd.StateCtx`)
	}

	return cmd.StateCtx, nil
}

func GetStateByLabels(stateCtx *StateCtx, labels map[string]string) *GetStateByLabelsCommand {
	return &GetStateByLabelsCommand{
		Labels:   labels,
		StateCtx: stateCtx,
	}
}

type GetStateByLabelsCommand struct {
	command

	Labels map[string]string

	StateCtx *StateCtx
}

func (cmd *GetStateByLabelsCommand) Result() (*StateCtx, error) {
	if cmd.StateCtx == nil {
		return nil, fmt.Errorf(`cmd.StateCtx is nil`)
	}

	return cmd.StateCtx, nil
}
