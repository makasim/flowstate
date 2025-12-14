//go:build goexperiment.synctest

package netflow_test

import (
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/netflow"
	"github.com/thejerf/slogassert"
)

func TestRegistry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lh := slogassert.New(t, slog.LevelDebug, nil)
		l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

		d := memdriver.New(l)

		firstFR := netflow.NewRegistry(`http://first:8080`, d, l)
		defer firstFR.Close()

		secondFR := netflow.NewRegistry(`http://second:8080`, d, l)
		defer secondFR.Close()

		// make sure we skiped the first round of Registry.watchFlows call
		time.Sleep(time.Second * 5)

		if err := firstFR.SetFlow(`aFlowOnFirstFR`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ *flowstate.Engine) (flowstate.Command, error) {
			panic("should not be called")
			return nil, nil
		})); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if err := secondFR.SetFlow(`aFlowOnSecondFR`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ *flowstate.Engine) (flowstate.Command, error) {
			panic("should not be called")
			return nil, nil
		})); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		time.Sleep(time.Second * 7)

		// now we should be able to get all flows

		f0, err := firstFR.Flow(`aFlowOnFirstFR`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if f0 == nil {
			t.Fatalf("expected non-nil flow, got nil")
		}
		if _, ok := f0.(flowstate.FlowFunc); !ok {
			t.Fatalf("expected flow to be of type FlowFunc, got %T", f0)
		}

		f0, err = firstFR.Flow(`aFlowOnSecondFR`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if f0 == nil {
			t.Fatalf("expected non-nil flow, got nil")
		}
		if _, ok := f0.(*netflow.Flow); !ok {
			t.Fatalf("expected flow to be of type FlowFunc, got %T", f0)
		}

		f0, err = secondFR.Flow(`aFlowOnSecondFR`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if f0 == nil {
			t.Fatalf("expected non-nil flow, got nil")
		}
		if _, ok := f0.(flowstate.FlowFunc); !ok {
			t.Fatalf("expected flow to be of type FlowFunc, got %T", f0)
		}

		f0, err = secondFR.Flow(`aFlowOnFirstFR`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if f0 == nil {
			t.Fatalf("expected non-nil flow, got nil")
		}
		if _, ok := f0.(*netflow.Flow); !ok {
			t.Fatalf("expected flow to be of type FlowFunc, got %T", f0)
		}

		// unset flows

		if err := firstFR.UnsetFlow(`aFlowOnFirstFR`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if err := firstFR.UnsetFlow(`aFlowOnSecondFR`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if err := secondFR.UnsetFlow(`aFlowOnFirstFR`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if err := secondFR.UnsetFlow(`aFlowOnSecondFR`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// wait for watchFlows to sync
		time.Sleep(time.Second * 12)

		// no flows should be available now

		if _, err = firstFR.Flow(`aFlowOnFirstFR`); !errors.Is(err, flowstate.ErrFlowNotFound) {
			t.Fatalf("expected flow not found error, got %v", err)
		}
		if _, err = firstFR.Flow(`aFlowOnSecondFR`); !errors.Is(err, flowstate.ErrFlowNotFound) {
			t.Fatalf("expected flow not found error, got %v", err)
		}
		if _, err = secondFR.Flow(`aFlowOnFirstFR`); !errors.Is(err, flowstate.ErrFlowNotFound) {
			t.Fatalf("expected flow not found error, got %v", err)
		}
		if _, err = secondFR.Flow(`aFlowOnSecondFR`); !errors.Is(err, flowstate.ErrFlowNotFound) {
			t.Fatalf("expected flow not found error, got %v", err)
		}
	})
}

func TestRegistry_Flow_SlowPath(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lh := slogassert.New(t, slog.LevelDebug, nil)
		l := slog.New(slogassert.New(t, slog.LevelDebug, lh))
		l = slog.Default()

		d := memdriver.New(l)

		fr := netflow.NewRegistry(`http://aHost:8080`, d, l)
		defer fr.Close()

		// make sure we skipped the first round of Registry.watchFlows call
		time.Sleep(time.Second * 5)

		// simulate a flow registered by someone else, but not yet available in the registry
		stateCtx := &flowstate.StateCtx{
			Current: flowstate.State{
				ID: `flowstate.flow.aFlowID`,
			},
		}
		stateCtx.Current.SetLabel(`flow.type`, `remote`)
		stateCtx.Current.SetAnnotation(`flowstate.flow.transition_id`, `aFlowID`)
		stateCtx.Current.SetAnnotation(`flowstate.flow.http_host`, `http://anotherHost:8080`)
		if err := d.Commit(flowstate.Commit(flowstate.Park(stateCtx))); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if _, err := fr.Flow(`aFlowID`); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
