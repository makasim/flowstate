package testcases

import (
	"strconv"
	"testing"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	//"go.uber.org/goleak"
)

func Cron(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	trkr := &Tracker{}

	mustSetFlow(fr, "cron", flowstate.FlowFunc(func(cronStateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(cronStateCtx, trkr)

		now := time.Now()

		cron, err := cronexpr.Parse(cronStateCtx.Current.Annotations[`cron`])
		if err != nil {
			cronStateCtx.Current.SetAnnotation(`error`, err.Error())
			return flowstate.Commit(
				flowstate.Park(cronStateCtx),
			), nil
		}
		if cron.Next(now).IsZero() {
			cronStateCtx.Current.SetAnnotation(`error`, `next time is zero`)
			return flowstate.Commit(
				flowstate.Park(cronStateCtx),
			), nil
		}

		taskFlowID := cronStateCtx.Current.Annotations[`cron_task_flow_id`]
		if taskFlowID == "" {
			cronStateCtx.Current.SetAnnotation(`error`, `taskFlowID is empty`)
			return flowstate.Commit(
				flowstate.Park(cronStateCtx),
			), nil
		}

		nextTimes := cron.NextN(now, 2)

		// can run task right now ?
		if nextTimes[0].After(now) && nextTimes[0].Before(now.Add(time.Second)) {
			taskStateCtx := &flowstate.StateCtx{
				Current: flowstate.State{
					ID: flowstate.StateID("task_" + strconv.FormatInt(nextTimes[0].Unix(), 10)),
					Annotations: map[string]string{
						"cron":      cronStateCtx.Current.Annotations[`cron`],
						"cron_task": "true",
					},
				},
			}

			if err := e.Do(
				flowstate.Commit(
					flowstate.Park(cronStateCtx),
					flowstate.DelayUntil(cronStateCtx, `cron`, nextTimes[1]),
					flowstate.Transit(taskStateCtx, flowstate.FlowID(taskFlowID)),
				),
				flowstate.Execute(taskStateCtx),
			); err != nil {
				return nil, err
			}

			return flowstate.Noop(), nil
		}

		if err := e.Do(flowstate.Commit(
			flowstate.Park(cronStateCtx),
			flowstate.DelayUntil(cronStateCtx, `cron`, nextTimes[0]),
		)); err != nil {
			return nil, err
		}

		return flowstate.Noop(), nil
	}))
	mustSetFlow(fr, "task", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.Park(stateCtx),
		), nil
	}))

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "aTID",
			Annotations: map[string]string{
				"cron":              "* * * * * * *",
				"cron_task_flow_id": "task",
			},
		},
	}

	require.NoError(t, e.Do(
		flowstate.Commit(
			flowstate.Transit(stateCtx, `cron`),
		),
		flowstate.Execute(stateCtx),
	))

	trkr.WaitVisitedEqual(t, []string{
		`cron`,
		`task`,
		`cron`,
		`task`,
		`cron`,
		`task`,
	}, time.Second*20)
}
