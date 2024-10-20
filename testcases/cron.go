package testcases

import (
	"context"
	"strconv"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func Cron(t TestingT, d flowstate.Doer, fr FlowRegistry) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{}

	fr.SetFlow("cron", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		now := time.Now()

		cron, err := cronexpr.Parse(stateCtx.Current.Annotations[`cron`])
		if err != nil {
			stateCtx.Current.SetAnnotation(`error`, err.Error())
			return flowstate.Commit(
				flowstate.End(stateCtx),
			), nil
		}
		if cron.Next(now).IsZero() {
			stateCtx.Current.SetAnnotation(`error`, `next time is zero`)
			return flowstate.Commit(
				flowstate.End(stateCtx),
			), nil
		}

		taskFlowID := stateCtx.Current.Annotations[`cron_task_flow_id`]
		if taskFlowID == "" {
			stateCtx.Current.SetAnnotation(`error`, `taskFlowID is empty`)
			return flowstate.Commit(
				flowstate.End(stateCtx),
			), nil
		}

		nextTimes := cron.NextN(now, 2)

		// can run task right now ?
		if nextTimes[0].After(now) && nextTimes[0].Before(now.Add(time.Second)) {
			taskStateCtx := &flowstate.StateCtx{
				Current: flowstate.State{
					ID: flowstate.StateID("task_" + strconv.FormatInt(nextTimes[0].Unix(), 10)),
					Annotations: map[string]string{
						"cron":      stateCtx.Current.Annotations[`cron`],
						"cron_task": "true",
					},
				},
			}

			if err := e.Do(
				flowstate.Commit(
					flowstate.Pause(stateCtx),
					flowstate.Delay(stateCtx, nextTimes[1].Sub(now)),
					flowstate.Transit(taskStateCtx, flowstate.FlowID(taskFlowID)),
				),
				flowstate.Execute(taskStateCtx),
			); err != nil {
				return nil, err
			}

			return flowstate.Noop(stateCtx), nil
		}
		
		if err := e.Do(flowstate.Commit(
			flowstate.Pause(stateCtx),
			flowstate.Delay(stateCtx, nextTimes[0].Sub(now)),
		)); err != nil {
			return nil, err
		}

		return flowstate.Noop(stateCtx), nil
	}))
	fr.SetFlow("task", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e *flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	e, err := flowstate.NewEngine(d)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

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
	}, time.Second*10)
}
