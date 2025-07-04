package testcases

import (
	"testing"

	"github.com/makasim/flowstate"
)

type TestingT interface {
	Helper()
	Error(...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...any)
	FailNow()
	Cleanup(f func())
}

type Suite struct {
	SetUp func(t TestingT) (flowstate.Driver, FlowRegistry)

	cases map[string]func(t TestingT, d flowstate.Driver, fr FlowRegistry)
}

func (s *Suite) Test(main *testing.T) {
	for name, fn := range s.cases {
		main.Run(name, func(t *testing.T) {
			if fn == nil {
				t.SkipNow()
			}

			d, fr := s.SetUp(t)
			fn(t, d, fr)
		})
	}
}

func (s *Suite) Skip(t *testing.T, name string) {
	if _, ok := s.cases[name]; !ok {
		t.Fatal("unknown test case: ", name)
	}

	s.cases[name] = nil
}

func Get(setUp func(t TestingT) (flowstate.Driver, FlowRegistry)) *Suite {
	return &Suite{
		SetUp: setUp,

		cases: map[string]func(t TestingT, d flowstate.Driver, fr FlowRegistry){
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
