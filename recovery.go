package flowstate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

var RecoveryEnabledAnnotation = `flowstate.recovery.enabled`

func DisableRecovery(stateCtx *StateCtx) {
	stateCtx.Current.SetAnnotation(RecoveryEnabledAnnotation, "false")
}

func recoveryEnabled(state State) bool {
	return state.Annotations[RecoveryEnabledAnnotation] != "false"
}

var RecoveryAttemptAnnotation = `flowstate.recovery.attempt`

func RecoveryAttempt(state State) int {
	attempt, _ := strconv.Atoi(state.Transition.Annotations[RecoveryAttemptAnnotation])
	return attempt
}

func setRecoveryAttempt(stateCtx *StateCtx, attempt int) {
	stateCtx.Current.Transition.SetAnnotation(RecoveryAttemptAnnotation, strconv.Itoa(attempt))
}

var DefaultMaxRecoveryAttempts = 3
var MaxRecoveryAttemptsAnnotation = `flowstate.recovery.max_attempts`

func MaxRecoveryAttempts(state State) int {
	attempt, _ := strconv.Atoi(state.Annotations[MaxRecoveryAttemptsAnnotation])
	if attempt <= 0 {
		return DefaultMaxRecoveryAttempts
	}

	return attempt
}

func SetMaxRecoveryAttempts(stateCtx *StateCtx, attempts int) {
	if attempts <= 0 {
		attempts = DefaultMaxRecoveryAttempts
	}

	stateCtx.Current.SetAnnotation(MaxRecoveryAttemptsAnnotation, strconv.Itoa(attempts))
}

var DefaultRetryAfter = time.Minute * 2
var MinRetryAfter = time.Minute
var MaxRetryAfter = time.Minute * 5
var RetryAfterAnnotation = `flowstate.recovery.retry_after`

func SetRetryAfter(stateCtx *StateCtx, retryAfter time.Duration) {
	if retryAfter < MinRetryAfter {
		retryAfter = MinRetryAfter
	}
	if retryAfter > MaxRetryAfter {
		retryAfter = MaxRetryAfter
	}

	stateCtx.Current.SetAnnotation(RetryAfterAnnotation, retryAfter.String())
}

func retryAt(state State) time.Time {
	retryAfterStr := state.Annotations[RetryAfterAnnotation]
	if retryAfterStr == "" {
		retryAfterStr = DefaultRetryAfter.String()
	}

	retryAfter, err := time.ParseDuration(retryAfterStr)
	if err != nil {
		return state.CommittedAt().Add(MinRetryAfter)
	}

	if retryAfter < MinRetryAfter {
		retryAfter = MinRetryAfter
	}
	if retryAfter > MaxRetryAfter {
		retryAfter = MaxRetryAfter
	}

	return state.CommittedAt().Add(retryAfter)
}

var recoveryStateID = StateID(`flowstate.recovery.meta`)

type Recoverer struct {
	mux sync.Mutex

	recoveryStateCtx *StateCtx

	active   bool
	sinceRev int64
	headRev  int64
	headTime time.Time
	tailRev  int64
	tailTime time.Time

	states               map[StateID]retryableState
	statesMaxSize        int
	statesMaxTailHeadDur time.Duration

	added     int64
	completed int64
	retried   int64
	dropped   int64
	commited  int64

	e         Engine
	stopCh    chan struct{}
	stoppedCh chan struct{}
	l         *slog.Logger
}

func NewRecoverer(e Engine, l *slog.Logger) *Recoverer {
	return &Recoverer{
		e: e,
		l: l,

		statesMaxSize:        100000,
		statesMaxTailHeadDur: MaxRetryAfter * 2,

		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

func (r *Recoverer) Init() error {
	recoverStateCtx := &StateCtx{}
	active := true
	if err := r.e.Do(GetStateByID(recoverStateCtx, recoveryStateID, 0)); errors.Is(err, ErrNotFound) {
		recoverStateCtx = &StateCtx{
			Current: State{
				ID:  recoveryStateID,
				Rev: 0,
			},
		}
		DisableRecovery(recoverStateCtx)
		setRecoverySinceRev(recoverStateCtx, 0)

		if err := r.e.Do(Commit(
			Transit(recoverStateCtx, `na`),
		)); IsErrRevMismatch(err) {
			// another process is already doing recovery, we can continue in standby mode
			active = false
		} else if err != nil {
			return fmt.Errorf("commit recovery state: %w", err)
		}
		r.commited++
	} else if err != nil {
		return fmt.Errorf("get recovery state: %w", err)
	} else {
		active = Paused(recoverStateCtx.Current)
	}

	r.reset(recoverStateCtx, active)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.updateHead()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		r.updateTail()
	}()
	go func() {
		wg.Wait()
		close(r.stoppedCh)
	}()

	return nil
}

func (r *Recoverer) Shutdown(ctx context.Context) error {
	close(r.stopCh)

	select {
	case <-r.stoppedCh:
		setRecoverySinceRev(r.recoveryStateCtx, r.nextSinceRev())
		if err := r.e.Do(Commit(Pause(r.recoveryStateCtx))); IsErrRevMismatch(err) {
			return nil
		} else if err != nil {
			return fmt.Errorf("commit: pause recovery state: %w", err)
		}

		r.reset(r.recoveryStateCtx, false)

		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type RecovererStats struct {
	HeadRev  int64
	HeadTime time.Time
	TailRev  int64
	TailTime time.Time

	Commited  int64
	Added     int64
	Completed int64
	Retried   int64
	Dropped   int64

	Active bool
}

func (r *Recoverer) Stats() RecovererStats {
	r.mux.Lock()
	defer r.mux.Unlock()

	return RecovererStats{
		HeadRev:  r.headRev,
		HeadTime: r.headTime,
		TailRev:  r.tailRev,
		TailTime: r.tailTime,

		Added:     r.added,
		Completed: r.completed,
		Retried:   r.retried,
		Dropped:   r.dropped,
		Commited:  r.commited,

		Active: r.active,
	}
}

func (r *Recoverer) updateHead() {
	t := time.NewTicker(time.Second * 10)
	defer t.Stop()

	prevAt := time.Now()
	var dur time.Duration
	for {
		if err := r.doUpdateHead(dur); err != nil {
			r.l.Error(fmt.Sprintf("update head: %s; retrying", err))
			continue
		}

		select {
		case <-r.stopCh:
			return
		case at := <-t.C:
			dur = at.Sub(prevAt)
			prevAt = at
		}
	}
}

func (r *Recoverer) doUpdateHead(dur time.Duration) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	first := true
	for {
		// TODO: breaks ticks logic
		//if len(r.states) > r.statesMaxSize {
		//	r.headTime = r.headTime.Add(dur)
		//	return nil
		//}
		//if len(r.states) > 0 && r.tailTime.Add(r.statesMaxTailHeadDur).Before(r.headTime) {
		//	r.headTime = r.headTime.Add(dur)
		//	return nil
		//}

		getManyCmd := GetStatesByLabels(nil).WithSinceRev(r.sinceRev)
		if err := r.e.Do(getManyCmd); err != nil {
			return fmt.Errorf("get many states: %w; since_rev=%d", err, r.sinceRev)
		}
		res, err := getManyCmd.Result()
		if err != nil {
			return fmt.Errorf("get many states: result: %w", err)
		}

		completed, added := r.completed, r.added
		for _, state := range res.States {
			r.sinceRev = state.Rev

			if state.ID == recoveryStateID {
				r.recoveryStateCtx = state.CopyToCtx(r.recoveryStateCtx)
				if state.Rev > r.recoveryStateCtx.Committed.Rev {
					active := Paused(state)
					r.reset(r.recoveryStateCtx, active)
				}
				continue
			}

			if !recoveryEnabled(state) {
				continue
			}

			r.headRev = state.Rev
			r.headTime = state.CommittedAt()
			if len(r.states) == 0 {
				r.tailRev = r.headRev
				r.tailTime = r.headTime
			}

			if !r.active {
				continue
			}
			if Ended(state) || Paused(state) {
				delete(r.states, state.ID)
				r.completed++

				continue
			}

			r.states[state.ID] = retryableState{
				State:   state.CopyTo(&State{}),
				retryAt: retryAt(state),
			}
			r.added++
		}

		if first && r.completed == completed && r.added == added {
			r.headTime = r.headTime.Add(dur)
			if len(r.states) == 0 {
				r.tailRev = r.headRev
				r.tailTime = r.headTime
			}

			if len(res.States) == 0 {
				return nil
			}
		}

		first = false

		if res.More {
			r.mux.Unlock()
			r.mux.Lock()
			continue
		}

		return nil
	}
}

func (r *Recoverer) updateTail() {
	t := time.NewTicker(time.Second * 10)
	defer t.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-t.C:
			if err := r.doUpdateTail(); err != nil {
				r.l.Error(fmt.Sprintf("update tail: %s; retrying", err))
				continue
			}
		}
	}
}

func (r *Recoverer) doUpdateTail() error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if !r.active {
		commitedAt := r.recoveryStateCtx.Committed.CommittedAt()
		if (commitedAt.Add(MaxRetryAfter+time.Minute).Before(time.Now()) && r.nextSinceRev() > getRecoverySinceRev(r.recoveryStateCtx)) ||
			Paused(r.recoveryStateCtx.Current) {
			nextRecoveryStateCtx := r.recoveryStateCtx.CopyTo(&StateCtx{})
			if err := r.e.Do(Commit(Resume(r.recoveryStateCtx))); IsErrRevMismatch(err) {
				r.reset(r.recoveryStateCtx, false)
				return nil
			} else if err != nil {
				return fmt.Errorf("commit recovery state: %w", err)
			}
			r.commited++
			r.recoveryStateCtx = nextRecoveryStateCtx.CopyTo(r.recoveryStateCtx)

			r.reset(r.recoveryStateCtx, true)
		}

		return nil
	}

	now := time.Now()

	if err := r.doRetry(); err != nil {
		return fmt.Errorf("do retry: %w", err)
	}

	tailRev := r.headRev
	var tailTime time.Time
	for _, retState := range r.states {
		if retState.Rev < tailRev {
			tailRev = retState.Rev
			tailTime = retState.CommittedAt()
		}
	}
	r.tailRev = tailRev
	r.tailTime = tailTime

	commitedAt := r.recoveryStateCtx.Committed.CommittedAt()
	if r.tailRev > getRecoverySinceRev(r.recoveryStateCtx)+1000 ||
		(commitedAt.Add(MaxRetryAfter).Before(now) && r.nextSinceRev() > getRecoverySinceRev(r.recoveryStateCtx)) {
		nextStateCtx := r.recoveryStateCtx.CopyTo(&StateCtx{})

		setRecoverySinceRev(nextStateCtx, r.nextSinceRev())
		if err := r.e.Do(Commit(Resume(nextStateCtx))); IsErrRevMismatch(err) {
			r.reset(r.recoveryStateCtx, false)
			return nil
		} else if err != nil {
			return fmt.Errorf("commit recovery state: %w", err)
		}

		r.commited++
		r.recoveryStateCtx = nextStateCtx.CopyTo(r.recoveryStateCtx)
	}

	return nil
}

func (r *Recoverer) doRetry() error {
	if len(r.states) == 0 {
		return nil
	}

	states := make([]State, 0)

	for id, retState := range r.states {
		if retState.retryAt.After(r.headTime) {
			continue
		}

		states = append(states, retState.State.CopyTo(&State{}))
		delete(r.states, id)
	}

	for _, state := range states {
		maxAttempts := MaxRecoveryAttempts(state)
		attempt := RecoveryAttempt(state) + 1
		stateCtx := state.CopyToCtx(&StateCtx{})

		setRecoveryAttempt(stateCtx, attempt)
		if attempt > maxAttempts {
			if err := r.e.Do(Commit(End(stateCtx))); IsErrRevMismatch(err) {
				continue
			} else if err != nil {
				return fmt.Errorf("commit state %s:%d reached max retry attempts %d and forcfully ended: %s", state.ID, state.Rev, maxAttempts, err)
			}

			r.dropped++
			continue
		}

		if err := r.e.Do(
			Commit(CommitStateCtx(stateCtx)),
			Execute(stateCtx),
		); IsErrRevMismatch(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("commit state %s:%d recovery attempt %d: %s", state.ID, state.Rev, attempt, err)
		}

		r.retried++
	}

	return nil
}

func (r *Recoverer) reset(recoveryStateCtx *StateCtx, active bool) {
	r.active = active
	r.recoveryStateCtx = recoveryStateCtx.CopyTo(&StateCtx{})

	r.sinceRev = getRecoverySinceRev(r.recoveryStateCtx)
	r.headRev = r.sinceRev
	r.headTime = time.Time{}
	r.tailRev = r.headRev
	r.tailTime = time.Time{}

	r.states = make(map[StateID]retryableState)
}

func (r *Recoverer) nextSinceRev() int64 {
	if r.tailRev == 0 {
		return 0
	}

	return r.tailRev - 1
}

func getRecoverySinceRev(stateCtx *StateCtx) int64 {
	sinceRevStr := stateCtx.Current.Annotations[`flowstate.recovery.since_rev`]
	if sinceRevStr == "" {
		return 0
	}

	sinceRev, _ := strconv.ParseInt(sinceRevStr, 10, 64)
	return sinceRev
}

func setRecoverySinceRev(stateCtx *StateCtx, sinceRev int64) {
	stateCtx.Current.SetAnnotation(`flowstate.recovery.since_rev`, strconv.FormatInt(sinceRev, 10))
}

type retryableState struct {
	State
	retryAt time.Time
}
