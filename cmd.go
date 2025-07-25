package flowstate

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var _ Command = &TransitCommand{}

var _ Command = &PauseCommand{}

var _ Command = &ResumeCommand{}

var _ Command = &ParkCommand{}

var _ Command = &DelayCommand{}

var _ Command = &NoopCommand{}

var _ Command = &StackCommand{}

var _ Command = &UnstackCommand{}

var _ Command = &GetStateByIDCommand{}

var _ Command = &GetStateByLabelsCommand{}

var _ Command = &GetStatesCommand{}

var _ Command = &GetDelayedStatesCommand{}

var _ Command = &AttachDataCommand{}

var _ Command = &GetDataCommand{}

var _ Command = &CommitCommand{}

var _ Command = &CommitStateCtxCommand{}

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

func CommitStateCtx(stateCtx *StateCtx) *CommitStateCtxCommand {
	return &CommitStateCtxCommand{
		StateCtx: stateCtx,
	}
}

type CommitStateCtxCommand struct {
	command
	StateCtx *StateCtx
}

func (cmd *CommitStateCtxCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func Parked(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `ended`
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

func (cmd *ParkCommand) Do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := nextTransitionOrCurrent(cmd.StateCtx, ``)
	for k, v := range cmd.Annotations {
		nextTs.SetAnnotation(k, v)
	}
	nextTs.SetAnnotation(StateAnnotation, `ended`)
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

func Noop(stateCtx *StateCtx) *NoopCommand {
	return &NoopCommand{
		StateCtx: stateCtx,
	}
}

type NoopCommand struct {
	command
	StateCtx *StateCtx
}

func Paused(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `paused`
}

func Pause(stateCtx *StateCtx) *PauseCommand {
	return &PauseCommand{
		StateCtx: stateCtx,
		To:       stateCtx.Current.Transition.To,
	}
}

type PauseCommand struct {
	command

	StateCtx *StateCtx
	To       FlowID
}

func (cmd *PauseCommand) WithTransit(to FlowID) *PauseCommand {
	cmd.To = to
	return cmd
}

func (cmd *PauseCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func (cmd *PauseCommand) Do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := nextTransitionOrCurrent(cmd.StateCtx, cmd.To)
	nextTs.SetAnnotation(StateAnnotation, `paused`)
	cmd.StateCtx.Current.Transition = nextTs

	return nil
}

func Resumed(state State) bool {
	return state.Transition.Annotations[StateAnnotation] == `resumed`
}

func Resume(stateCtx *StateCtx) *ResumeCommand {
	return &ResumeCommand{
		StateCtx: stateCtx,
	}
}

type ResumeCommand struct {
	command
	StateCtx *StateCtx
	To       FlowID
}

func (cmd *ResumeCommand) WithTransit(to FlowID) *ResumeCommand {
	cmd.To = to
	return cmd
}

func (cmd *ResumeCommand) CommittableStateCtx() *StateCtx {
	return cmd.StateCtx
}

func (cmd *ResumeCommand) Do() error {
	cmd.StateCtx.Transitions = append(cmd.StateCtx.Transitions, cmd.StateCtx.Current.Transition)

	nextTs := nextTransitionOrCurrent(cmd.StateCtx, cmd.To)
	nextTs.SetAnnotation(StateAnnotation, `resumed`)
	cmd.StateCtx.Current.Transition = nextTs

	return nil
}

var StateAnnotation = `flowstate.state`

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

func AttachData(stateCtx *StateCtx, data *Data, alias string) *AttachDataCommand {
	return &AttachDataCommand{
		StateCtx: stateCtx,
		Data:     data,
		Alias:    alias,

		Store: true,
	}
}

type AttachDataCommand struct {
	command
	StateCtx *StateCtx
	Data     *Data
	Alias    string
	Store    bool
}

func (cmd *AttachDataCommand) WithoutStore() *AttachDataCommand {
	cmd.Store = false
	return cmd
}

func (cmd *AttachDataCommand) Prepare() error {
	if cmd.Alias == "" {
		return fmt.Errorf("alias is empty")
	}
	if cmd.Data.ID == "" {
		cmd.Data.ID = DataID(ulid.Make().String())
	}
	if cmd.Data.Rev < 0 {
		return fmt.Errorf("Data.Rev is negative")
	}
	if cmd.Data.B == nil || len(cmd.Data.B) == 0 {
		return fmt.Errorf("Data.B is empty")
	}
	if cmd.Data.Rev == 0 && !cmd.Store {
		return fmt.Errorf("Data.Rev is zero, but Store is false; this would lead to data loss")
	}

	return nil
}

func (cmd *AttachDataCommand) Do() {
	cmd.StateCtx.Current.SetAnnotation(
		dataAnnotation(cmd.Alias),
		string(cmd.Data.ID)+":"+strconv.FormatInt(cmd.Data.Rev, 10),
	)
}

func GetData(stateCtx *StateCtx, data *Data, alias string) *GetDataCommand {
	return &GetDataCommand{
		StateCtx: stateCtx,
		Data:     data,
		Alias:    alias,
	}

}

type GetDataCommand struct {
	command
	StateCtx *StateCtx
	Data     *Data
	Alias    string
}

func (cmd *GetDataCommand) Prepare() error {
	if cmd.Data == nil {
		return fmt.Errorf("data is nil")
	}
	if cmd.Alias == "" {
		return fmt.Errorf("alias is empty")
	}

	annotKey := dataAnnotation(cmd.Alias)
	idRevStr := cmd.StateCtx.Current.Annotations[annotKey]
	if idRevStr == "" {
		return fmt.Errorf("annotation %q is not set", annotKey)
	}

	sepIdx := strings.LastIndexAny(idRevStr, ":")
	if sepIdx < 1 || sepIdx+1 == len(idRevStr) {
		return fmt.Errorf("annotation %q contains invalid data reference; got %q", annotKey, idRevStr)
	}

	id := DataID(idRevStr[:sepIdx])
	rev, err := strconv.ParseInt(idRevStr[sepIdx+1:], 10, 64)
	if err != nil {
		return fmt.Errorf("annotation %q contains invalid data revision; got %q: %w", annotKey, idRevStr[sepIdx+1:], err)
	}

	cmd.Data.ID = id
	cmd.Data.Rev = rev

	return nil
}

func dataAnnotation(alias string) string {
	return "flowstate.data." + string(alias)
}

func nextTransitionOrCurrent(stateCtx *StateCtx, to FlowID) Transition {
	if to == `` {
		to = stateCtx.Current.Transition.To
	}

	return Transition{
		To: to,
	}
}
