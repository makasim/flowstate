//go:build goexperiment.synctest

package flowstate_test

import (
	"context"
	"log/slog"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/thejerf/slogassert"
)

func TestIter(t *testing.T) {
	f := func(genFn func(d flowstate.Driver), cmd *flowstate.GetStatesCommand, exp int) {
		t.Helper()

		synctest.Test(t, func(t *testing.T) {
			t.Helper()

			lh := slogassert.New(t, slog.LevelDebug, nil)
			l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

			d := memdriver.New(l)

			go genFn(d)
			synctest.Wait()

			time.Sleep(time.Minute)
			synctest.Wait()

			iter := flowstate.NewIter(d, cmd)

			stopT := time.NewTimer(time.Minute * 5)
			var act int
		loop:
			for {
				select {
				case <-stopT.C:
					break loop
				default:
				}

				for iter.Next() {
					act++
				}
				if iter.Err() != nil {
					t.Fatalf("unexpected error from iter: %v", iter.Err())
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
				iter.Wait(ctx)
				cancel()
			}

			if act != exp {
				t.Errorf("expected to read %d states; got %d", exp, act)
			}
		})
	}

	gen100 := func(d flowstate.Driver) {
		for i := 0; i < 100; i++ {
			labels := map[string]string{}
			if i%2 == 0 {
				labels["bar"] = "barVal"
			}
			if i%3 == 0 {
				labels["foo"] = "fooVal"
			}

			if err := d.Commit(flowstate.Commit(flowstate.Park(&flowstate.StateCtx{
				Current: flowstate.State{
					ID:     flowstate.StateID("state" + strconv.Itoa(i)),
					Labels: labels,
				},
			}))); err != nil {
				panic(err)
			}
		}
	}

	// no states
	f(func(d flowstate.Driver) {}, flowstate.GetStatesByLabels(nil), 0)

	// iterate over existing states
	f(gen100, flowstate.GetStatesByLabels(nil), 100)

	// iterate one by one
	f(gen100, flowstate.GetStatesByLabels(nil).WithLimit(1), 100)

	// iterate by 2
	f(gen100, flowstate.GetStatesByLabels(nil).WithLimit(2), 100)

	// iterate by 99
	f(gen100, flowstate.GetStatesByLabels(nil).WithLimit(99), 100)

	// iterate by 100
	f(gen100, flowstate.GetStatesByLabels(nil).WithLimit(100), 100)

	// iterate by 101
	f(gen100, flowstate.GetStatesByLabels(nil).WithLimit(101), 100)

	// iterate over foo=fooVal states
	f(gen100, flowstate.GetStatesByLabels(map[string]string{"foo": "fooVal"}), 34)

	// iterate over foo=fooVal states
	f(gen100, flowstate.GetStatesByLabels(map[string]string{"bar": "barVal"}), 50)

	// states are added every 10s while iterating
	f(func(d flowstate.Driver) {
		for i := 0; i < 3; i++ {
			for j := 0; j < 10; j++ {
				if err := d.Commit(flowstate.Commit(flowstate.Park(&flowstate.StateCtx{
					Current: flowstate.State{
						ID: flowstate.StateID("state-" + strconv.Itoa(i) + "-" + strconv.Itoa(j)),
					},
				}))); err != nil {
					panic(err)
				}
			}
			time.Sleep(time.Second * 10)
		}
	}, flowstate.GetStatesByLabels(nil), 30)

	// states are added every minute while iterating
	f(func(d flowstate.Driver) {
		for i := 0; i < 3; i++ {
			for j := 0; j < 10; j++ {
				if err := d.Commit(flowstate.Commit(flowstate.Park(&flowstate.StateCtx{
					Current: flowstate.State{
						ID: flowstate.StateID("state-" + strconv.Itoa(i) + "-" + strconv.Itoa(j)),
					},
				}))); err != nil {
					panic(err)
				}
			}
			time.Sleep(time.Minute)
		}
	}, flowstate.GetStatesByLabels(nil), 30)

	// states are added only after some time
	f(func(d flowstate.Driver) {
		time.Sleep(time.Minute * 2)
		for j := 0; j < 10; j++ {
			if err := d.Commit(flowstate.Commit(flowstate.Park(&flowstate.StateCtx{
				Current: flowstate.State{
					ID: flowstate.StateID("state-" + strconv.Itoa(j)),
				},
			}))); err != nil {
				panic(err)
			}
		}
	}, flowstate.GetStatesByLabels(nil), 10)
}
