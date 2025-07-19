//go:build goexperiment.synctest

package flowstate_test

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/thejerf/slogassert"
)

func TestDelayer(t *testing.T) {
	f := func(delayingStates []delayingStateFunc, exp []delayedState) {
		t.Helper()

		synctest.Run(func() {
			lh := slogassert.New(t, slog.LevelDebug, nil)
			l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

			actMux := &sync.Mutex{}
			act := make([]delayedState, 0, len(delayingStates))
			d := memdriver.New(l)
			fr := &flowstate.DefaultFlowRegistry{}
			mustSetFlow(fr, `delayed`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
				actMux.Lock()
				defer actMux.Unlock()

				act = append(act, delayedState{
					StateID: stateCtx.Current.ID,
					At:      time.Now().UTC().Truncate(time.Second * 10),
				})

				return flowstate.Commit(flowstate.End(stateCtx)), nil
			}))

			e, err := flowstate.NewEngine(d, fr, l)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			defer func() {
				if err := e.Shutdown(context.Background()); err != nil {
					t.Fatalf("failed to shutdown engine: %v", err)
				}
			}()

			var wg sync.WaitGroup
			for _, ds := range delayingStates {
				ds := ds
				wg.Add(1)
				go func() {
					defer wg.Done()

					ds(t, e)
				}()
			}

			time.Sleep(time.Minute * 10)
			synctest.Wait()

			dlr, err := flowstate.NewDelayer(e, l)
			if err != nil {
				t.Fatalf("failed to create delayer: %v", err)
			}
			defer func() {
				if err := dlr.Shutdown(context.Background()); err != nil {
					t.Fatalf("failed to shutdown delayer: %v", err)
				}
			}()

			time.Sleep(time.Minute * 20)
			synctest.Wait()

			wg.Wait()

			sort.Slice(exp, func(i, j int) bool {
				return exp[i].StateID < exp[j].StateID
			})
			sort.Slice(act, func(i, j int) bool {
				return act[i].StateID < act[j].StateID
			})

			if !reflect.DeepEqual(exp, act) {
				t.Fatalf("expected delayed states %v, got %v", exp, act)
			}
		})
	}

	// delayed states added before the delayer starts
	f(
		[]delayingStateFunc{
			sleepAndDelay(flowstate.State{ID: `s1`}, 0, time.Minute),
			sleepAndDelay(flowstate.State{ID: `s2`}, 0, time.Minute*2),
			sleepAndDelay(flowstate.State{ID: `s3`}, 0, time.Minute*3),
		},
		[]delayedState{
			{`s1`, parseTime(`2000-01-01T00:10:00Z`)},
			{`s2`, parseTime(`2000-01-01T00:10:00Z`)},
			{`s3`, parseTime(`2000-01-01T00:10:00Z`)},
		},
	)

	// delayed states added during the delayer is running, the delay time is in future
	f(
		[]delayingStateFunc{
			sleepAndDelay(flowstate.State{ID: `s1`}, time.Minute*14, time.Minute),
			sleepAndDelay(flowstate.State{ID: `s2`}, time.Minute*15, time.Minute),
			sleepAndDelay(flowstate.State{ID: `s3`}, time.Minute*16, time.Minute),
		},
		[]delayedState{
			{`s1`, parseTime(`2000-01-01T00:15:00Z`)},
			{`s2`, parseTime(`2000-01-01T00:16:00Z`)},
			{`s3`, parseTime(`2000-01-01T00:17:00Z`)},
		},
	)

	// delayed states added during the delayer is running, the delay time is in past
	f(
		[]delayingStateFunc{
			sleepAndDelay(flowstate.State{ID: `s1`}, time.Minute*14, -time.Minute*5),
			sleepAndDelay(flowstate.State{ID: `s2`}, time.Minute*15, -time.Minute*5),
			sleepAndDelay(flowstate.State{ID: `s3`}, time.Minute*16, -time.Minute*5),
		},
		[]delayedState{
			{`s1`, parseTime(`2000-01-01T00:14:00Z`)},
			{`s2`, parseTime(`2000-01-01T00:15:00Z`)},
			{`s3`, parseTime(`2000-01-01T00:16:00Z`)},
		},
	)

	// delayed states added during the delayer is running, the delay time is mixed
	f(
		[]delayingStateFunc{
			sleepAndDelay(flowstate.State{ID: `s1`}, time.Minute*15, time.Minute*5),
			sleepAndDelay(flowstate.State{ID: `s2`}, time.Minute*15, time.Minute*5),
			sleepAndDelay(flowstate.State{ID: `s3`}, time.Minute*15, -time.Minute*5),
		},
		[]delayedState{
			{`s1`, parseTime(`2000-01-01T00:20:00Z`)},
			{`s2`, parseTime(`2000-01-01T00:20:00Z`)},
			{`s3`, parseTime(`2000-01-01T00:15:00Z`)},
		},
	)

	// delayed states too far in the future
	f(
		[]delayingStateFunc{
			sleepAndDelay(flowstate.State{ID: `s1`}, 0, time.Minute*31),
			sleepAndDelay(flowstate.State{ID: `s2`}, 0, time.Minute*31),
			sleepAndDelay(flowstate.State{ID: `s3`}, 0, time.Minute*31),
		},
		[]delayedState{},
	)

	// delay committed state
	f(
		[]delayingStateFunc{
			func(t *testing.T, e flowstate.Engine) {
				stateCtx := &flowstate.StateCtx{
					Current: flowstate.State{ID: `s1`},
				}
				if err := e.Do(flowstate.Commit(flowstate.Transit(stateCtx, `aFlow`))); err != nil {
					t.Fatalf("failed to commit state: %v", err)
				}

				stateCtx.Current.Transition.To = `delayed`

				if err := e.Do(flowstate.Delay(stateCtx, time.Minute*15)); err != nil {
					t.Fatalf("failed to delay state: %v", err)
				}
			},
		},
		[]delayedState{
			{StateID: `s1`, At: parseTime(`2000-01-01T00:15:00Z`)},
		},
	)

	// ignore on rev mismatch
	f(
		[]delayingStateFunc{
			func(t *testing.T, e flowstate.Engine) {
				stateCtx := &flowstate.StateCtx{
					Current: flowstate.State{ID: `s1`},
				}
				stateCtx.Current.Transition.To = `delayed`

				if err := e.Do(flowstate.Commit(flowstate.CommitStateCtx(stateCtx))); err != nil {
					t.Fatalf("failed to commit state: %v", err)
				}

				stateCtx1 := stateCtx.CopyTo(&flowstate.StateCtx{})
				if err := e.Do(flowstate.Commit(flowstate.CommitStateCtx(stateCtx1))); err != nil {
					t.Fatalf("failed to commit state: %v", err)
				}

				if err := e.Do(flowstate.Delay(stateCtx, time.Minute*15)); err != nil {
					t.Fatalf("failed to delay state: %v", err)
				}
			},
		},
		[]delayedState{},
	)

	// with, a commit it causes a rev mismatch, without commit it should execute state
	f(
		[]delayingStateFunc{
			func(t *testing.T, e flowstate.Engine) {
				stateCtx := &flowstate.StateCtx{
					Current: flowstate.State{ID: `s1`},
				}
				stateCtx.Current.Transition.To = `delayed`

				if err := e.Do(flowstate.Commit(flowstate.CommitStateCtx(stateCtx))); err != nil {
					t.Fatalf("failed to commit state: %v", err)
				}

				stateCtx1 := stateCtx.CopyTo(&flowstate.StateCtx{})
				if err := e.Do(flowstate.Commit(flowstate.CommitStateCtx(stateCtx1))); err != nil {
					t.Fatalf("failed to commit state: %v", err)
				}

				if err := e.Do(flowstate.Delay(stateCtx, time.Minute*15).WithCommit(false)); err != nil {
					t.Fatalf("failed to delay state: %v", err)
				}
			},
		},
		[]delayedState{
			{StateID: `s1`, At: parseTime(`2000-01-01T00:15:00Z`)},
		},
	)
}

func TestDelayer_Concurrency(t *testing.T) {
	synctest.Run(func() {
		lh := slogassert.New(t, slog.LevelDebug, nil)
		l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

		var delayedFutureCnt atomic.Int64
		var delayedPastCnt atomic.Int64
		d := memdriver.New(l)
		fr := &flowstate.DefaultFlowRegistry{}
		mustSetFlow(fr, `delayed`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
			if stateCtx.Current.Labels[`delay`] == `future` {
				delayedFutureCnt.Add(1)
			} else {
				delayedPastCnt.Add(1)
			}

			return flowstate.Commit(flowstate.End(stateCtx)), nil
		}))

		e, err := flowstate.NewEngine(d, fr, l)
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}
		defer e.Shutdown(context.Background())

		dlr, err := flowstate.NewDelayer(e, l)
		if err != nil {
			t.Fatalf("failed to create delayer: %v", err)
		}
		defer dlr.Shutdown(context.Background())

		var wg sync.WaitGroup

		// start some goroutines to delay states in future (>1min) at random intervals
		var delayingFutureCnt atomic.Int64
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < 50; j++ {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID(`future-` + strconv.Itoa(i) + "-" + strconv.Itoa(j)),
							Labels: map[string]string{
								`delay`: `future`,
							},
							Transition: flowstate.Transition{
								To: `delayed`,
							},
						},
					}

					if err := e.Do(flowstate.Delay(stateCtx, time.Minute)); err != nil {
						t.Fatalf("failed to delay state: %v", err)
					}

					delayingFutureCnt.Add(1)
					time.Sleep(time.Second * time.Duration(10+i+j))
				}
			}()
		}

		// start some goroutines to delay states in the past (<1min) at random intervals
		var delayingPastCnt atomic.Int64
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < 50; j++ {
					stateCtx := &flowstate.StateCtx{
						Current: flowstate.State{
							ID: flowstate.StateID(`past-` + strconv.Itoa(i) + "-" + strconv.Itoa(j)),
							Labels: map[string]string{
								`delay`: `past`,
							},
							Transition: flowstate.Transition{
								To: `delayed`,
							},
						},
					}

					if err := e.Do(flowstate.Delay(stateCtx, -time.Second*10)); err != nil {
						t.Fatalf("failed to delay state: %v", err)
					}

					delayingPastCnt.Add(1)
					<-time.NewTimer(time.Second * time.Duration(10+i+j)).C
				}
			}()
		}

		wg.Wait()

		time.Sleep(time.Minute * 40)

		if err := dlr.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown delayer: %v", err)
		}
		if err := e.Shutdown(context.Background()); err != nil {
			t.Fatalf("failed to shutdown engine: %v", err)
		}

		// guard
		if delayingFutureCnt.Load() == 0 {
			t.Fatalf("expected some delayed future states, got none")
		}
		if delayedFutureCnt.Load() != delayingFutureCnt.Load() {
			t.Fatalf("expected %d delayed future states, got %d", delayingFutureCnt.Load(), delayedFutureCnt.Load())
		}

		// guard
		if delayingPastCnt.Load() == 0 {
			t.Fatalf("expected some delayed past states, got none")
		}
		if delayedPastCnt.Load() != delayingPastCnt.Load() {
			t.Fatalf("expected %d delayed past states, got %d", delayingPastCnt.Load(), delayedPastCnt.Load())
		}
	})
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

type delayingStateFunc func(t *testing.T, e flowstate.Engine)

func sleepAndDelay(state flowstate.State, sleepDur, dur time.Duration) delayingStateFunc {
	return func(t *testing.T, e flowstate.Engine) {
		stateCtx := state.CopyToCtx(&flowstate.StateCtx{})
		stateCtx.Current.Transition.To = `delayed`

		time.Sleep(sleepDur)

		if err := e.Do(flowstate.Delay(stateCtx, dur)); err != nil {
			t.Fatalf("failed to delay state: %v", err)
		}
	}
}

type delayedState struct {
	StateID flowstate.StateID
	At      time.Time
}

func mustSetFlow(fr flowstate.FlowRegistry, id flowstate.TransitionID, f flowstate.Flow) {
	if err := fr.SetFlow(id, f); err != nil {
		panic(fmt.Sprintf("set flow %s: %s", id, err))
	}
}
