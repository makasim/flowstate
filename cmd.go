package flowstate

import (
	"encoding/base64"
	"fmt"
	"time"
)

var _ Command = &TransitCommand{}

var _ Command = &ParkCommand{}

var _ Command = &DelayCommand{}

var _ Command = &NoopCommand{}

var _ Command = &StackCommand{}

var _ Command = &UnstackCommand{}

var _ Command = &GetStateByIDCommand{}

var _ Command = &GetStateByLabelsCommand{}

var _ Command = &GetStatesCommand{}

var _ Command = &GetDelayedStatesCommand{}

var _ Command = &CommitCommand{}

var _ Command = &ExecuteCommand{}

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

func Commit(cmds ...Command) *CommitCommand {
	return &CommitCommand{
		Commands: cmds,
	}
}

type CommittableCommand interface {
	CommittableStateCtx() *StateCtx
}

type CommitCommand struct {
	command
	Commands []Command
}

func (cmd *CommitCommand) setSessID(id int64) {
	cmd.command.setSessID(id)
	for _, subCmd := range cmd.Commands {
		subCmd.setSessID(id)
	}
}

func Parked(state State) bool {
	return state.Transition.To == ``
}

func Park(stateCtx *StateCtx) *ParkCommand {
	return &ParkCommand{
		StateCtx: stateCtx,
	}
}

type ParkCommand struct {
	command
	StateCtx    *StateCtx
	Annotations map[string]string
}

func (cmd *ParkCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func (cmd *ParkCommand) WithAnnotation(name, value string) *ParkCommand {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}

	cmd.Annotations[name] = value
	return cmd
}

func (cmd *ParkCommand) WithAnnotations(annotations map[string]string) *ParkCommand {
	for k, v := range annotations {
		cmd.WithAnnotation(k, v)
	}
	return cmd
}

func (cmd *ParkCommand) Do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := Transition{}
	for k, v := range cmd.Annotations {
		nextTs.SetAnnotation(k, v)
	}
	cmd.StateCtx.Current.Transition = nextTs

	return nil
}

func Execute(stateCtx *StateCtx) *ExecuteCommand {
	return &ExecuteCommand{
		StateCtx: stateCtx,
	}
}

type ExecuteCommand struct {
	command
	StateCtx *StateCtx

	sync bool
}

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

func (cmd *GetStateByIDCommand) Prepare() error {
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

const GetStatesDefaultLimit = 50

type GetStatesResult struct {
	States []State
	More   bool
}

func GetStatesByLabels(labels map[string]string) *GetStatesCommand {
	return (&GetStatesCommand{
		Limit: GetStatesDefaultLimit,
	}).WithORLabels(labels)
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

// WithSinceRev sets SinceRev filter for the command.
// States with revision greater than SinceRev will be returned.
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

func (cmd *GetStatesCommand) Prepare() {
}

func Noop() *NoopCommand {
	return &NoopCommand{}
}

type NoopCommand struct {
	command
}

func Transit(stateCtx *StateCtx, to FlowID) *TransitCommand {
	return &TransitCommand{
		StateCtx: stateCtx,
		To:       to,
	}
}

type TransitCommand struct {
	command
	StateCtx    *StateCtx
	Annotations map[string]string
	To          FlowID
}

func (cmd *TransitCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func (cmd *TransitCommand) WithAnnotation(name, value string) *TransitCommand {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}

	cmd.Annotations[name] = value
	return cmd
}

func (cmd *TransitCommand) WithAnnotations(annotations map[string]string) *TransitCommand {
	for k, v := range annotations {
		cmd.WithAnnotation(k, v)
	}
	return cmd
}

func (cmd *TransitCommand) Do() error {
	if cmd.To == "" {
		return fmt.Errorf("flow id empty")
	}

	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := nextTransitionOrCurrent(cmd.StateCtx, cmd.To)
	for k, v := range cmd.Annotations {
		nextTs.SetAnnotation(k, v)
	}
	cmd.StateCtx.Current.Transition = nextTs

	return nil
}

func Stack(carrierStateCtx, stackStateCtx *StateCtx, annotation string) *StackCommand {
	return &StackCommand{
		StackedStateCtx: stackStateCtx,
		CarrierStateCtx: carrierStateCtx,
		Annotation:      annotation,
	}

}

type StackCommand struct {
	command
	StackedStateCtx *StateCtx
	CarrierStateCtx *StateCtx
	Annotation      string
}

func (cmd *StackCommand) Do() error {
	if cmd.Annotation == `` {
		return fmt.Errorf("stack annotation name empty")
	}
	if cmd.CarrierStateCtx.Current.Annotations[cmd.Annotation] != `` {
		return fmt.Errorf("stack annotation '%s' already set", cmd.Annotation)
	}

	b := MarshalStateCtx(cmd.StackedStateCtx, nil)
	b64 := base64.StdEncoding.EncodeToString(b)
	cmd.CarrierStateCtx.Current.SetAnnotation(cmd.Annotation, b64)

	return nil
}

func Unstack(carrierStateCtx, unstackStateCtx *StateCtx, annotation string) *UnstackCommand {
	return &UnstackCommand{
		CarrierStateCtx: carrierStateCtx,
		UnstackStateCtx: unstackStateCtx,
		Annotation:      annotation,
	}

}

type UnstackCommand struct {
	command
	CarrierStateCtx *StateCtx
	UnstackStateCtx *StateCtx
	Annotation      string
}

func (cmd *UnstackCommand) Do() error {
	b64 := cmd.CarrierStateCtx.Current.Annotations[cmd.Annotation]
	if b64 == `` {
		return fmt.Errorf("stack annotation value empty")
	}

	b, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return fmt.Errorf("cannot base64 decode state in annotation '%s': %s", cmd.Annotation, err)
	}

	if err := UnmarshalStateCtx(b, cmd.UnstackStateCtx); err != nil {
		return fmt.Errorf("cannot unmarshal stack in annotation '%s': %s", cmd.Annotation, err)
	}

	cmd.CarrierStateCtx.Current.Annotations[cmd.Annotation] = ``

	return nil
}

func StoreData(stateCtx *StateCtx, alias string) *StoreDataCommand {
	return &StoreDataCommand{
		StateCtx: stateCtx,
		Alias:    alias,
	}
}

type StoreDataCommand struct {
	command
	StateCtx *StateCtx
	Alias    string
}

func (cmd *StoreDataCommand) Prepare() (bool, error) {
	d, err := cmd.StateCtx.Data(cmd.Alias)
	if err != nil {
		return false, err
	}

	if d.Rev < 0 {
		return false, fmt.Errorf("data rev is negative")
	}

	if !d.isDirty() {
		referenceData(cmd.StateCtx, cmd.Alias, d.Rev)
		return false, nil
	}

	d.checksum()

	return true, nil
}

func (cmd *StoreDataCommand) post() {
	d := cmd.StateCtx.MustData(cmd.Alias)
	referenceData(cmd.StateCtx, cmd.Alias, d.Rev)
}

func GetData(stateCtx *StateCtx, alias string) *GetDataCommand {
	return &GetDataCommand{
		StateCtx: stateCtx,
		Alias:    alias,
	}

}

type GetDataCommand struct {
	command
	StateCtx *StateCtx
	Alias    string
}

func (cmd *GetDataCommand) Prepare() (bool, error) {
	rev, err := dereferenceData(cmd.StateCtx, cmd.Alias)
	if err != nil {
		return false, err
	}

	d, err := cmd.StateCtx.Data(cmd.Alias)
	if err != nil {
		cmd.StateCtx.SetData(cmd.Alias, &Data{
			Rev: rev,
		})
		return true, nil
	} else if d.Rev == rev {
		return false, nil
	}

	d.Rev = rev
	d.Blob = d.Blob[:0]
	for k := range d.Annotations {
		delete(d.Annotations, k)
	}

	return true, nil
}

func nextTransitionOrCurrent(stateCtx *StateCtx, to FlowID) Transition {
	if to == `` {
		to = stateCtx.Current.Transition.To
	}

	return Transition{
		To: to,
	}
}
