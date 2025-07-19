package testcases

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"go.uber.org/goleak"
)

type Suite struct {
	SetUp        func(t *testing.T) flowstate.Driver
	SetUpDelayer bool

	disableGoleak bool
	cases         map[string]func(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver)
}

func (s *Suite) Test(main *testing.T) {
	for name := range s.cases {
		s.run(main, name)
	}
}

func (s *Suite) run(main *testing.T, name string) {
	if !s.disableGoleak {
		defer goleak.VerifyNone(main, goleak.IgnoreCurrent())
	}

	main.Run(name, func(t *testing.T) {
		t.Helper()

		fn := s.cases[name]
		if fn == nil {
			t.SkipNow()
		}

		l, _ := NewTestLogger(t)

		d := s.SetUp(t)
		fr := &flowstate.DefaultFlowRegistry{}

		e, err := flowstate.NewEngine(d, fr, l)
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			if err := e.Shutdown(ctx); err != nil {
				t.Fatalf("failed to shutdown engine: %v", err)
			}
		})

		if s.SetUpDelayer {
			dlr, err := flowstate.NewDelayer(e, l)
			if err != nil {
				t.Fatalf("failed to create delayer: %v", err)
			}
			t.Cleanup(func() {
				sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
				defer sCtxCancel()

				if err := dlr.Shutdown(sCtx); err != nil {
					t.Fatalf("failed to shutdown delayer: %v", err)
				}
			})
		}

		fn(t, e, fr, d)
	})
}

func (s *Suite) DisableGoleak() {
	s.disableGoleak = true
}

func (s *Suite) Skip(t *testing.T, name string) {
	if _, ok := s.cases[name]; !ok {
		t.Fatal("unknown test case: ", name)
	}

	s.cases[name] = nil
}

func Get(setUp func(t *testing.T) flowstate.Driver) *Suite {
	return &Suite{
		SetUp:        setUp,
		SetUpDelayer: true,

		cases: map[string]func(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver){
			"Actor": Actor,

			"CallFlow":           CallFlow,
			"CallFlowWithCommit": CallFlowWithCommit,
			"CallFlowWithWatch":  CallFlowWithWatch,

			"Condition": Condition,

			"DataFlowConfig":         DataFlowConfig,
			"DataStoreGet":           DataStoreGet,
			"DataStoreGetWithCommit": DataStoreGetWithCommit,

			"Delay": Delay,

			"Fork":              Fork,
			"ForkJoinFirstWins": ForkJoin_FirstWins,
			"ForkJoinLastWins":  ForkJoin_LastWins,
			"ForkWithCommit":    Fork_WithCommit,

			"GetOneByIDAndRev":    GetOneByIDAndRev,
			"GetOneLatestByID":    GetOneLatestByID,
			"GetOneLatestByLabel": GetOneLatestByLabel,
			"GetOneNotFound":      GetOneNotFound,

			"GetManyLabels":      GetManyLabels,
			"GetManyOrLabels":    GetManyORLabels,
			"GetManySinceLatest": GetManySinceLatest,
			"GetManySinceRev":    GetManySinceRev,
			"GetManySinceTime":   GetManySinceTime,
			"GetManyLatestOnly":  GetManyLatestOnly,

			"Mutex":     Mutex,
			"Queue":     Queue,
			"RateLimit": RateLimit,

			"SingleNode":                   SingleNode,
			"ThreeConsequentNodes":         ThreeConsequentNodes,
			"TwoConsequentNodes":           TwoConsequentNodes,
			"TwoConsequentNodesWithCommit": TwoConsequentNodesWithCommit,

			"WatchLabels":      WatchLabels,
			"WatchOrLabels":    WatchORLabels,
			"WatchSinceLatest": WatchSinceLatest,
			"WatchSinceRev":    WatchSinceRev,
			"WatchSinceTime":   WatchSinceTime,

			"Cron": Cron,
		},
	}
}

func filterSystemStates(states []flowstate.State) []flowstate.State {
	res := make([]flowstate.State, 0, len(states))
	for _, s := range states {
		if strings.HasPrefix(string(s.ID), "flowstate.") {
			continue
		}

		res = append(res, s)
	}

	return res
}
