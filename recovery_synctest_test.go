//go:build goexperiment.synctest

package flowstate_test

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/thejerf/slogassert"
)

func TestRecovererRetryLogic(t *testing.T) {
	f := func(genFn func(e flowstate.Engine), flowFn flowstate.FlowFunc, expStats flowstate.RecovererStats) {
		t.Helper()

		synctest.Test(t, func(t *testing.T) {
			t.Helper()

			lh := slogassert.New(t, slog.LevelDebug, nil)
			l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

			d := memdriver.New(l)
			fr := &flowstate.DefaultFlowRegistry{}
			mustSetFlow(fr, `aFlow`, flowFn)

			e, err := flowstate.NewEngine(d, fr, l)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			r, err := flowstate.NewRecoverer(e, l)
			if err != nil {
				t.Fatalf("failed to create recoverer: %v", err)
			}

			go genFn(e)
			synctest.Wait()

			time.Sleep(time.Hour)
			synctest.Wait()

			if err := r.Shutdown(context.Background()); err != nil {
				t.Fatalf("failed to shutdown recoverer: %v", err)
			}
			if err := e.Shutdown(context.Background()); err != nil {
				t.Fatalf("failed to shutdown engine: %v", err)
			}
			time.Sleep(time.Minute)
			synctest.Wait()

			actStats := r.Stats()
			if expStats.Added != actStats.Added {
				t.Errorf("expected added equal to %d; got %d", expStats.Added, actStats.Added)
			}
			if expStats.Completed != actStats.Completed {
				t.Errorf("expected completed equal to %d; got %d", expStats.Completed, actStats.Completed)
			}
			if expStats.Retried != actStats.Retried {
				t.Errorf("expected retried equal to %d; got %d", expStats.Retried, actStats.Retried)
			}
			if expStats.Dropped != actStats.Dropped {
				t.Errorf("expected dropped equal to %d; got %d", expStats.Dropped, actStats.Dropped)
			}
			if expStats.Commited != actStats.Commited {
				t.Errorf("expected commited equal to %d; got %d", expStats.Commited, actStats.Commited)
			}
		})
	}

	// recoverer should commit recoveryStateCtx only once if there is no state commits.
	f(
		func(e flowstate.Engine) {},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			t.Fatalf("unexpected call to flow function")
			return nil, nil
		},
		flowstate.RecovererStats{
			Commited: 1,
		},
	)

	// states with disabled recovery ignored completely
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 100; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
							Transition: flowstate.Transition{
								To: `aFlow`,
							},
						},
					}
					flowstate.DisableRecovery(stateCtx)

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}
					if err := e.Execute(stateCtx); err != nil {
						t.Fatalf("failed to execute state %s: %v", stateCtx.Current.ID, err)
					}
				}()
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Commited: 1,
		},
	)

	// parked states marked completed
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 100; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Park(stateCtx))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}
				}()
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			t.Fatalf("unexpected call to flow function")
			return nil, nil
		},
		flowstate.RecovererStats{
			Completed: 100,
			Commited:  2,
		},
	)

	// a state retried if there is no any further commits (aka test ticks work)
	f(
		func(e flowstate.Engine) {
			stateCtx := &flowstate.StateCtx{
				Current: flowstate.State{
					ID: flowstate.StateID("aTID"),
				},
			}

			if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
				t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
			}

			// just commit, do not execute, recovery should kick in
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     2,
			Completed: 1,
			Retried:   1,
			Commited:  2,
		},
	)

	// every fifth state retried should be retried
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 100; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					if i%5 != 0 {
						if err := e.Execute(stateCtx); err != nil {
							t.Fatalf("failed to execute state %s: %v", stateCtx.Current.ID, err)
						}
						return
					}
					// just commit, do not execute, recovery should kick in
				}()
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     120,
			Completed: 100,
			Retried:   20,
			Commited:  2,
		},
	)

	// a state retried default max times (3)
	f(
		func(e flowstate.Engine) {
			stateCtx := &flowstate.StateCtx{
				Current: flowstate.State{
					ID: flowstate.StateID("aTID"),
				},
			}

			if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
				t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
			}

			// just commit, do not execute, recovery should kick in
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			// do nothing to see how many times the state will be retried
			return flowstate.Noop(), nil
		},
		flowstate.RecovererStats{
			Added:     4,
			Completed: 1,
			Retried:   3,
			Dropped:   1,
			Commited:  3,
		},
	)

	// a state retried custom set max times (4)
	f(
		func(e flowstate.Engine) {
			stateCtx := &flowstate.StateCtx{
				Current: flowstate.State{
					ID: flowstate.StateID("aTID"),
				},
			}
			flowstate.SetMaxRecoveryAttempts(stateCtx, 4)

			if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
				t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
			}

			// just commit, do not execute, recovery should kick in
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			// do nothing to see how many times the state will be retried
			return flowstate.Noop(), nil
		},
		flowstate.RecovererStats{
			Added:     5,
			Completed: 1,
			Retried:   4,
			Dropped:   1,
			Commited:  4,
		},
	)

	// a state retried if there are new commits
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 30; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					// just commit, do not execute, recovery should kick in
				}()

				time.Sleep(time.Second * 30)
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     60,
			Completed: 30,
			Retried:   30,
			Commited:  5,
		},
	)

	// test that recovery state commited when rev difference > 1000
	f(
		func(e flowstate.Engine) {
			var wg sync.WaitGroup
			for i := 0; i < 1500; i++ {
				if i%100 == 0 {
					wg.Wait()
				}

				wg.Add(1)
				go func() {
					defer wg.Done()

					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					if err := e.Execute(stateCtx); err != nil {
						t.Fatalf("failed to execute state %s: %v", stateCtx.Current.ID, err)
					}
				}()
			}

			wg.Wait()
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     1500,
			Completed: 1500,
			Commited:  2,
		},
	)

	// default retry after (2m) is respected - successful execution
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 10; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					time.Sleep(time.Second * 110)

					if err := e.Execute(stateCtx); err != nil {
						t.Fatalf("failed to execute state %s: %v", stateCtx.Current.ID, err)
					}
				}()
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     10,
			Completed: 10,
			Commited:  2,
		},
	)

	// default retry after (2m) is respected - retry execution
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 10; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					time.Sleep(time.Second * 150)

					if err := e.Execute(stateCtx); err != nil {
						t.Fatalf("failed to execute state %s: %v", stateCtx.Current.ID, err)
					}
				}()
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     20,
			Completed: 10,
			Retried:   10,
			Commited:  2,
		},
	)

	// custom retry after (1m) is respected - successful execution
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 10; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}
					flowstate.SetRetryAfter(stateCtx, time.Minute)

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					time.Sleep(time.Second * 50)

					if err := e.Execute(stateCtx); err != nil {
						t.Fatalf("failed to execute state %s: %v", stateCtx.Current.ID, err)
					}
				}()
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     10,
			Completed: 10,
			Commited:  2,
		},
	)

	// custom retry after (1m) is respected - retry execution
	f(
		func(e flowstate.Engine) {
			for i := 0; i < 10; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}
					flowstate.SetRetryAfter(stateCtx, time.Minute)

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					time.Sleep(time.Second * 90)

					if err := e.Execute(stateCtx); err != nil {
						t.Fatalf("failed to execute state %s: %v", stateCtx.Current.ID, err)
					}
				}()
			}
		},
		func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		},
		flowstate.RecovererStats{
			Added:     20,
			Completed: 10,
			Retried:   10,
			Commited:  2,
		},
	)
}

// This test simulates a cluster of two recoverers, one active and one standby.
// The active recoverer processes states and the standby recoverer takes over if the active one is shut down.
// The test ensures that all states are retried and processed correctly,
// and that the standby recoverer becomes active when the active one is shut down.
func TestRecovererActiveStandby(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lh := slogassert.New(t, slog.LevelDebug, nil)
		l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

		d := memdriver.New(l)
		fr := &flowstate.DefaultFlowRegistry{}
		mustSetFlow(fr, `aFlow`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		}))

		e, err := flowstate.NewEngine(d, fr, l)
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}

		r0, err := flowstate.NewRecoverer(e, l)
		if err != nil {
			t.Fatalf("failed to create r0 recoverer: %v", err)
		}

		r1, err := flowstate.NewRecoverer(e, l)
		if err != nil {
			t.Fatalf("failed to create r1 recoverer: %v", err)
		}

		if !r0.Stats().Active {
			t.Fatalf("expected recoverer 0 to be active, but it is not")
		}
		if r1.Stats().Active {
			t.Fatalf("expected recoverer 1 to be standby, but it is active")
		}

		go func() {
			for i := 0; i < 30; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					// just commit, do not execute, recovery should kick in
				}()

				time.Sleep(time.Second * 30)
			}
		}()

		time.Sleep(time.Minute * 4)

		if err := r0.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown recoverer 0: %v", err)
		}

		time.Sleep(time.Second * 30)

		if r0.Stats().Active {
			t.Fatalf("expected recoverer 0 to be standby, but it is active")
		}
		if !r1.Stats().Active {
			t.Fatalf("expected recoverer 1 to be active, but it is not")
		}

		time.Sleep(time.Hour)

		if err := r1.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown recoverer 1: %v", err)
		}
		if err := e.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown engine: %v", err)
		}
		time.Sleep(time.Minute)

		r0Stats := r0.Stats()
		r1Stats := r1.Stats()
		if 30 != r0Stats.Retried+r1Stats.Retried {
			t.Errorf("expected retried equal to %d; got %d", 30, r0Stats.Retried+r1Stats.Retried)
		}
	})
}

// This test simulates a crash of the active recoverer and checks if the standby recoverer becomes active.
// The test ensures that the standby recoverer takes over and retries the states correctly.
func TestRecovererCrashStandbyBecomeActive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lh := slogassert.New(t, slog.LevelDebug, nil)
		l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

		d := memdriver.New(l)
		fr := &flowstate.DefaultFlowRegistry{}
		mustSetFlow(fr, `aFlow`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		}))

		e, err := flowstate.NewEngine(d, fr, l)
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}

		// simulate a crash of the active recoverer
		// by commit state that suggest there a running recoverer
		recoverStateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: `flowstate.recovery.meta`,
			},
		}
		flowstate.DisableRecovery(recoverStateCtx)
		if err := e.Do(flowstate.Commit(
			flowstate.Park(recoverStateCtx).WithAnnotation(`state`, `active`),
		)); err != nil {
			t.Fatalf("failed to commit recovery state: %v", err)
		}

		r, err := flowstate.NewRecoverer(e, l)
		if err != nil {
			t.Fatalf("failed to create recoverer: %v", err)
		}

		if r.Stats().Active {
			t.Fatalf("expected recoverer to be standby, but it is active")
		}

		go func() {
			for i := 0; i < 30; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}

					// just commit, do not execute, recovery should kick in
				}()

				time.Sleep(time.Second * 30)
			}
		}()

		time.Sleep(flowstate.MaxRetryAfter + time.Second*80)
		synctest.Wait()

		if !r.Stats().Active {
			t.Fatalf("expected recoverer to be active, but it is not")
		}

		time.Sleep(time.Hour)
		synctest.Wait()

		if err := r.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown recoverer: %v", err)
		}
		if err := e.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown engine: %v", err)
		}
		time.Sleep(time.Minute)
		synctest.Wait()

		actStats := r.Stats()
		if 30 != actStats.Retried {
			t.Errorf("expected retried equal to %d; got %d", 30, actStats.Retried)
		}
	})
}

func TestRecovererOnlyOneActive(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		assertOneActive := func(rs []*flowstate.Recoverer) {
			var activeCnt int
			for _, r := range rs {
				if r.Stats().Active {
					activeCnt++
				}
			}
			if activeCnt != 1 {
				t.Fatalf("expected only one recoverer to be active, but got %d", activeCnt)
			}
		}

		lh := slogassert.New(t, slog.LevelDebug, nil)
		l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

		d := memdriver.New(l)
		fr := &flowstate.DefaultFlowRegistry{}
		mustSetFlow(fr, `aFlow`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			return flowstate.Commit(flowstate.Park(stateCtx)), nil
		}))

		e, err := flowstate.NewEngine(d, fr, l)
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}

		var rs []*flowstate.Recoverer
		for i := 0; i < 2; i++ {
			r, err := flowstate.NewRecoverer(e, l)
			if err != nil {
				t.Fatalf("failed to create recoverer: %v", err)
			}

			rs = append(rs, r)
		}

		assertOneActive(rs)

		go func() {
			for i := 0; i < 30; i++ {
				go func() {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID("aTID" + strconv.Itoa(i)),
						},
					}

					if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
						t.Fatalf("failed to commit state %s: %v", stateCtx.Current.ID, err)
					}
					e.Execute(stateCtx)
				}()
			}
		}()

		time.Sleep(time.Second * 30)
		synctest.Wait()
		assertOneActive(rs)

		time.Sleep(time.Minute * 10)
		synctest.Wait()
		assertOneActive(rs)

		for _, r := range rs {
			if err := r.Shutdown(context.Background()); err != nil {
				t.Fatalf("failed to shutdown recoverer: %v", err)
			}
		}
		if err := e.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown engine: %v", err)
		}
	})
}
