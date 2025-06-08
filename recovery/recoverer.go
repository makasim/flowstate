package recovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/makasim/flowstate"
)

var stateID = flowstate.StateID(`flowstate.recovery.state`)

type Recoverer struct {
	mux sync.Mutex

	recoveryStateCtx *flowstate.StateCtx

	active   bool
	sinceRev int64
	headRev  int64
	headTime time.Time
	tailRev  int64
	tailTime time.Time

	states               map[flowstate.StateID]retryableState
	statesMaxSize        int
	statesMaxTailHeadDur time.Duration

	added     int64
	completed int64
	retried   int64
	dropped   int64
	commited  int64

	e         flowstate.Engine
	doneCh    chan struct{}
	stoppedCh chan struct{}
	l         *slog.Logger
}

func New(e flowstate.Engine, l *slog.Logger) *Recoverer {
	return &Recoverer{
		e: e,
		l: l,

		statesMaxSize:        100000,
		statesMaxTailHeadDur: MaxRetryAfter * 2,

		doneCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}
}

func (r *Recoverer) Init() error {
	recoverStateCtx := &flowstate.StateCtx{}
	active := true
	if err := r.e.Do(flowstate.GetByID(recoverStateCtx, stateID, 0)); errors.Is(err, flowstate.ErrNotFound) {
		recoverStateCtx = &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  stateID,
				Rev: 0,
			},
		}
		Disable(recoverStateCtx)
		setSinceRev(recoverStateCtx, 0)

		if err := r.e.Do(flowstate.Commit(
			flowstate.Transit(recoverStateCtx, `na`),
		)); flowstate.IsErrRevMismatch(err) {
			// another process is already doing recovery, we can continue in standby mode
			active = false
		} else if err != nil {
			return fmt.Errorf("commit recovery state: %w", err)
		}
		r.commited++
	} else if err != nil {
		return fmt.Errorf("get recovery state: %w", err)
	} else {
		active = flowstate.Paused(recoverStateCtx.Current)
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
	close(r.doneCh)

	select {
	case <-r.stoppedCh:
		setSinceRev(r.recoveryStateCtx, r.nextSinceRev())
		if err := r.e.Do(flowstate.Commit(flowstate.Pause(r.recoveryStateCtx))); flowstate.IsErrRevMismatch(err) {
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

type Stats struct {
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

func (r *Recoverer) Stats() Stats {
	r.mux.Lock()
	defer r.mux.Unlock()

	return Stats{
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
		case <-r.doneCh:
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

		getManyCmd := flowstate.GetManyByLabels(nil).WithSinceRev(r.sinceRev)
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

			if state.ID == stateID {
				r.recoveryStateCtx = state.CopyToCtx(r.recoveryStateCtx)
				if state.Rev > r.recoveryStateCtx.Committed.Rev {
					active := flowstate.Paused(state)
					r.reset(r.recoveryStateCtx, active)
				}
				continue
			}

			if !enabled(state) {
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
			if flowstate.Ended(state) || flowstate.Paused(state) {
				delete(r.states, state.ID)
				r.completed++

				continue
			}

			r.states[state.ID] = retryableState{
				State:   state.CopyTo(&flowstate.State{}),
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
		case <-r.doneCh:
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
		if (commitedAt.Add(MaxRetryAfter+time.Minute).Before(time.Now()) && r.nextSinceRev() > getSinceRev(r.recoveryStateCtx)) ||
			flowstate.Paused(r.recoveryStateCtx.Current) {
			nextRecoveryStateCtx := r.recoveryStateCtx.CopyTo(&flowstate.StateCtx{})
			if err := r.e.Do(flowstate.Commit(flowstate.Resume(r.recoveryStateCtx))); flowstate.IsErrRevMismatch(err) {
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
	if r.tailRev > getSinceRev(r.recoveryStateCtx)+1000 ||
		(commitedAt.Add(MaxRetryAfter).Before(now) && r.nextSinceRev() > getSinceRev(r.recoveryStateCtx)) {
		nextStateCtx := r.recoveryStateCtx.CopyTo(&flowstate.StateCtx{})

		setSinceRev(nextStateCtx, r.nextSinceRev())
		if err := r.e.Do(flowstate.Commit(flowstate.Resume(nextStateCtx))); flowstate.IsErrRevMismatch(err) {
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

	states := make([]flowstate.State, 0)

	for id, retState := range r.states {
		if retState.retryAt.After(r.headTime) {
			continue
		}

		states = append(states, retState.State.CopyTo(&flowstate.State{}))
		delete(r.states, id)
	}

	for _, state := range states {
		maxAttempts := MaxAttempts(state)
		attempt := Attempt(state) + 1
		stateCtx := state.CopyToCtx(&flowstate.StateCtx{})

		setAttempt(stateCtx, attempt)
		if attempt > maxAttempts {
			if err := r.e.Do(flowstate.Commit(flowstate.End(stateCtx))); flowstate.IsErrRevMismatch(err) {
				continue
			} else if err != nil {
				return fmt.Errorf("commit state %s:%d reached max retry attempts %d and forcfully ended: %s", state.ID, state.Rev, maxAttempts, err)
			}

			r.dropped++
			continue
		}

		if err := r.e.Do(
			flowstate.Commit(flowstate.CommitStateCtx(stateCtx)),
			flowstate.Execute(stateCtx),
		); flowstate.IsErrRevMismatch(err) {
			continue
		} else if err != nil {
			return fmt.Errorf("commit state %s:%d recovery attempt %d: %s", state.ID, state.Rev, attempt, err)
		}

		r.retried++
	}

	return nil
}

func (r *Recoverer) reset(recoveryStateCtx *flowstate.StateCtx, active bool) {
	r.active = active
	r.recoveryStateCtx = recoveryStateCtx.CopyTo(&flowstate.StateCtx{})

	r.sinceRev = getSinceRev(r.recoveryStateCtx)
	r.headRev = r.sinceRev
	r.headTime = time.Time{}
	r.tailRev = r.headRev
	r.tailTime = time.Time{}

	r.states = make(map[flowstate.StateID]retryableState)
}

func (r *Recoverer) nextSinceRev() int64 {
	if r.tailRev == 0 {
		return 0
	}

	return r.tailRev - 1
}

func getSinceRev(stateCtx *flowstate.StateCtx) int64 {
	sinceRevStr := stateCtx.Current.Annotations[`flowstate.recovery.since_rev`]
	if sinceRevStr == "" {
		return 0
	}

	sinceRev, _ := strconv.ParseInt(sinceRevStr, 10, 64)
	return sinceRev
}

func setSinceRev(stateCtx *flowstate.StateCtx, sinceRev int64) {
	stateCtx.Current.SetAnnotation(`flowstate.recovery.since_rev`, strconv.FormatInt(sinceRev, 10))
}

type retryableState struct {
	flowstate.State
	retryAt time.Time
}
