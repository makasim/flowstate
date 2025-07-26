package flowstate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

var DelayUntilAnnotation = `flowstate.delay.until`
var DelayCommitAnnotation = `flowstate.delay.commit`

func Delayed(state State) bool {
	return state.Transition.Annotations[DelayUntilAnnotation] != ``
}

func DelayedUntil(state State) time.Time {
	until, _ := time.Parse(time.RFC3339, state.Transition.Annotations[DelayUntilAnnotation])
	return until
}

func Delay(stateCtx *StateCtx, to FlowID, dur time.Duration) *DelayCommand {
	return &DelayCommand{
		StateCtx:  stateCtx,
		ExecuteAt: time.Now().Add(dur),
		To:        to,
		Commit:    true,
	}
}

func DelayUntil(stateCtx *StateCtx, to FlowID, executeAt time.Time) *DelayCommand {
	return &DelayCommand{
		StateCtx:  stateCtx,
		ExecuteAt: executeAt,
		To:        to,
		Commit:    true,
	}
}

type DelayCommand struct {
	command
	StateCtx    *StateCtx
	ExecuteAt   time.Time
	Commit      bool
	To          FlowID
	Annotations map[string]string

	Result *DelayedState
}

func (cmd *DelayCommand) WithTransit(to FlowID) *DelayCommand {
	cmd.To = to
	return cmd
}

func (cmd *DelayCommand) WithAnnotation(name, value string) *DelayCommand {
	if cmd.Annotations == nil {
		cmd.Annotations = make(map[string]string)
	}

	cmd.Annotations[name] = value
	return cmd
}

func (cmd *DelayCommand) WithCommit(commit bool) *DelayCommand {
	cmd.Commit = commit
	return cmd
}

func (cmd *DelayCommand) Prepare() error {
	if cmd.To == `` {
		return fmt.Errorf("flow id empty")
	}

	cmd.Result = &DelayedState{
		State:     cmd.StateCtx.Current.CopyTo(&State{}),
		ExecuteAt: cmd.ExecuteAt,
	}

	nextTs := nextTransitionOrCurrent(cmd.StateCtx, cmd.To)
	for k, v := range cmd.Annotations {
		nextTs.SetAnnotation(k, v)
	}

	nextTs.SetAnnotation(DelayUntilAnnotation, cmd.ExecuteAt.Format(time.RFC3339))
	if !cmd.Commit {
		nextTs.SetAnnotation(DelayCommitAnnotation, `false`)
	}
	cmd.Result.State.Transition = nextTs

	return nil
}

func (cmd *DelayCommand) MustResult() DelayedState {
	if cmd.Result == nil {
		panic("FATAL: MustResult must be called after successful execution of the command; have you checked for errors?")
	}

	return *cmd.Result
}

type DelayedState struct {
	State     State
	Offset    int64
	ExecuteAt time.Time
}

func (ds DelayedState) MarshalJSON() ([]byte, error) {
	jsonDS := &jsonDelayedState{}
	jsonDS.fromDelayedState(&ds)
	return json.Marshal(jsonDS)
}

func (ds *DelayedState) UnmarshalJSON(data []byte) error {
	jsonDS := &jsonDelayedState{}
	if err := json.Unmarshal(data, jsonDS); err != nil {
		return err
	}
	return jsonDS.toDelayedState(ds)
}

const GetDelayedStatesDefaultLimit = 500

type GetDelayedStatesResult struct {
	States []DelayedState
	More   bool
}

func GetDelayedStates(since, until time.Time, offset int64) *GetDelayedStatesCommand {
	return &GetDelayedStatesCommand{
		Since:  since,
		Until:  until,
		Offset: offset,
		Limit:  GetDelayedStatesDefaultLimit,
	}
}

type GetDelayedStatesCommand struct {
	command

	Since time.Time
	Until time.Time
	// Offset is valid inside the since-until range.
	// Should be used to pagination results.
	Offset int64
	Limit  int

	Result *GetDelayedStatesResult
}

func (cmd *GetDelayedStatesCommand) MustResult() *GetDelayedStatesResult {
	if cmd.Result == nil {
		panic("FATAL: MustResult must be called after successful execution of the command; have you checked for errors?")
	}

	return cmd.Result
}

func (cmd *GetDelayedStatesCommand) Prepare() {
	if cmd.Limit == 0 {
		cmd.Limit = GetDelayedStatesDefaultLimit
	}
	if cmd.Until.IsZero() {
		cmd.Until = time.Now()
	}
}

type Delayer struct {
	e Engine

	metaStateCtx *StateCtx
	offset       int64
	since        time.Time
	until        time.Time
	limit        int

	commitSince  time.Time
	commitOffset int64

	delayedStates map[int64]DelayedState

	stopCh    chan struct{}
	stoppedCh chan struct{}
	l         *slog.Logger
}

func NewDelayer(e Engine, l *slog.Logger) (*Delayer, error) {
	d := &Delayer{
		e: e,
		l: l,

		delayedStates: make(map[int64]DelayedState),
		stopCh:        make(chan struct{}),
		stoppedCh:     make(chan struct{}),
	}

	metaStateCtx := &StateCtx{}
	if err := d.e.Do(GetStateByID(metaStateCtx, `flowstate.delayer.meta`, 0)); errors.Is(err, ErrNotFound) {
		metaStateCtx.Current = State{
			ID:  `flowstate.delayer.meta`,
			Rev: 0,
		}
		setDelayerMetaState(metaStateCtx, time.Unix(0, 0).UTC(), 0)

		if err := d.e.Do(Commit(
			Park(metaStateCtx),
		)); IsErrRevMismatch(err) {
			return nil, fmt.Errorf("another process is already doing delaying; exiting (todo: implement standby mode)")
		} else if err != nil {
			return nil, fmt.Errorf("commit meta state: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("get meta state: %w", err)
	}

	d.metaStateCtx = metaStateCtx
	d.commitSince, d.commitOffset = getDelayerMetaState(metaStateCtx)
	d.since, d.offset = d.commitSince, d.commitOffset

	go func() {
		defer close(d.stoppedCh)

		updateHeadT := time.NewTicker(time.Second * 30)
		defer updateHeadT.Stop()

		updateHeadFreshT := time.NewTicker(time.Second * 5)
		defer updateHeadFreshT.Stop()

		updateTailT := time.NewTicker(time.Second)
		defer updateTailT.Stop()

		commitT := time.NewTicker(time.Minute)
		defer commitT.Stop()

		for {
			select {
			case now := <-updateHeadT.C:
				until := now.Add(time.Minute)
				if _, err := d.queryDelayedStates(d.since, until, 0); err != nil {
					d.l.Error(fmt.Sprintf("query delayed from %s to %s, offset=%d: %s", d.since, until, 0, err))
				}
				d.since = until
			case now := <-updateHeadFreshT.C:
				var since time.Time
				if d.offset > 0 {
					since = now.Add(-time.Hour * 24)
				}
				until := now.Add(time.Minute)
				nextOffset, err := d.queryDelayedStates(since, until, d.offset)
				if err != nil {
					d.l.Error(fmt.Sprintf("query delayed from %s to %s, offset=%d: %s", since, until, d.offset, err))
				}
				d.offset = nextOffset
			case now := <-updateTailT.C:
				if err := d.updateTail(now); err != nil {
					d.l.Error(fmt.Sprintf("update tail: %s; retrying", err.Error()))
				}
			case <-commitT.C:
				d.maybeCommitMeta()
			case <-d.stopCh:
				d.maybeCommitMeta()

				return
			}
		}
	}()

	return d, nil
}

func (d *Delayer) maybeCommitMeta() {
	// no delayed states at all, no need to commit
	if d.commitSince.Equal(time.Unix(0, 0).UTC()) && d.commitOffset == 0 {
		return
	}

	committedSince, committedOffset := getDelayerMetaState(d.metaStateCtx)
	if d.commitSince.Equal(committedSince) && d.commitOffset == committedOffset {
		// no changes since last commit, no need to commit
		return
	}

	nextMetaState := d.metaStateCtx.CopyTo(&StateCtx{})
	setDelayerMetaState(nextMetaState, d.commitSince, d.commitOffset)
	if err := d.e.Do(Commit(Park(nextMetaState))); IsErrRevMismatch(err) {
		d.l.Warn("another process is already doing delaying; exiting (todo: implement standby mode)")
	} else if err != nil {
		d.l.Error(fmt.Sprintf("commit meta state: %s", err))
	}

	nextMetaState.CopyTo(d.metaStateCtx)
}

func (d *Delayer) queryDelayedStates(since, until time.Time, offset int64) (int64, error) {
	nextOffset := offset
	for {
		if len(d.delayedStates) > 1000 {
			return 0, nil
		}

		cmd := GetDelayedStates(since, until, offset)
		if err := d.e.Do(cmd); err != nil {
			return 0, fmt.Errorf("get delayed states: %w", err)
		}

		res := cmd.MustResult()
		if len(res.States) == 0 {
			return nextOffset, nil
		}

		for _, state := range res.States {
			d.delayedStates[state.Offset] = state
			nextOffset = max(nextOffset, state.Offset)
		}

		if res.More {
			continue
		}

		return nextOffset, nil
	}
}

func (d *Delayer) updateTail(now time.Time) error {
	commitSince := d.commitSince
	commitOffset := d.commitOffset
	for _, delayedState := range d.delayedStates {
		if delayedState.ExecuteAt.After(now) {
			continue
		}

		stateCtx := delayedState.State.CopyToCtx(&StateCtx{})
		commit := stateCtx.Current.Transition.Annotations[DelayCommitAnnotation] != `false`
		if commit {
			transitCmd := Transit(stateCtx, stateCtx.Current.Transition.To).
				WithAnnotations(stateCtx.Current.Transition.Annotations)

			if err := d.e.Do(Commit(transitCmd)); IsErrRevMismatch(err) {
				continue
			} else if err != nil {
				return fmt.Errorf("commit state ctx: id=%s rev=%d: %w", delayedState.State.ID, delayedState.State.Rev, err)
			}
		}

		delete(d.delayedStates, delayedState.Offset)

		// TODO: add concurrency control
		go func() {
			if err := d.e.Execute(stateCtx); err != nil && !commit {
				// delayed state is not so we warn about it, if commit recovery would kick in
				d.l.Warn(fmt.Sprintf("delayed uncommited state execution has failed; id=%s rev=%d: %s", delayedState.State.ID, delayedState.State.Rev, err.Error()))
			}
		}()

		if delayedState.ExecuteAt.Before(commitSince) {
			commitSince = delayedState.ExecuteAt
		}
		commitOffset = max(commitOffset, delayedState.Offset)
	}

	d.commitSince, d.commitOffset = commitSince, commitOffset

	return nil
}

func (d *Delayer) Shutdown(ctx context.Context) error {
	select {
	case <-d.stopCh:
		return fmt.Errorf(`already shutdown`)
	default:
		close(d.stopCh)

		select {
		case <-d.stoppedCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func setDelayerMetaState(metaStateCtx *StateCtx, since time.Time, offset int64) {
	metaStateCtx.Current.SetAnnotation(`flowstate.delayer.offset`, strconv.FormatInt(offset, 10))
	metaStateCtx.Current.SetAnnotation(`flowstate.delayer.since`, since.Format(time.RFC3339))
}

func getDelayerMetaState(metaStateCtx *StateCtx) (time.Time, int64) {
	offset0 := metaStateCtx.Current.Annotations[`flowstate.delayer.offset`]
	offset, err := strconv.ParseInt(offset0, 10, 64)
	if err != nil {
		panic(fmt.Errorf("cannot parse flowstate.delayer.offset='%s' into int64: %w", offset0, err))
	}

	since0 := metaStateCtx.Current.Annotations[`flowstate.delayer.since`]
	since, err := time.Parse(time.RFC3339, since0)
	if err != nil {
		panic(fmt.Errorf("cannot parse flowstate.delayer.since=%s into time.Time: %w", since0, err))
	}

	return since, offset
}
