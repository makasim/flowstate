package testcases

import (
	"context"
	"strconv"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	//"go.uber.org/goleak"
)

func Cron(t TestingT, d flowstate.Doer, fr FlowRegistry) {
	// does not work in flowstatesrv srvdriver
	// delayer related goroutines are started inside test by stoped with the app outside
	//	cron.go:130: found unexpected goroutines:
	//[Goroutine 155 in state select, with github.com/makasim/flowstate/memdriver.(*Delayer).Do.func1 on top of the stack:
	//	github.com/makasim/flowstate/memdriver.(*Delayer).Do.func1()
	//	/foo/flowstatesrv/vendor/github.com/makasim/flowstate/memdriver/delayer.go:59 +0x18c
	//	created by github.com/makasim/flowstate/memdriver.(*Delayer).Do in goroutine 32
	//	/foo/flowstatesrv/vendor/github.com/makasim/flowstate/memdriver/delayer.go:51 +0x100
	//	]
	// defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	trkr := &Tracker{}

	fr.SetFlow("cron", flowstate.FlowFunc(func(cronStateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(cronStateCtx, trkr)

		now := time.Now()

		cron, err := cronexpr.Parse(cronStateCtx.Current.Annotations[`cron`])
		if err != nil {
			cronStateCtx.Current.SetAnnotation(`error`, err.Error())
			return flowstate.Commit(
				flowstate.End(cronStateCtx),
			), nil
		}
		if cron.Next(now).IsZero() {
			cronStateCtx.Current.SetAnnotation(`error`, `next time is zero`)
			return flowstate.Commit(
				flowstate.End(cronStateCtx),
			), nil
		}

		taskFlowID := cronStateCtx.Current.Annotations[`cron_task_flow_id`]
		if taskFlowID == "" {
			cronStateCtx.Current.SetAnnotation(`error`, `taskFlowID is empty`)
			return flowstate.Commit(
				flowstate.End(cronStateCtx),
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
					flowstate.Pause(cronStateCtx),
					flowstate.Delay(cronStateCtx, nextTimes[1].Sub(now)),
					flowstate.Transit(taskStateCtx, flowstate.FlowID(taskFlowID)),
				),
				flowstate.Execute(taskStateCtx),
			); err != nil {
				return nil, err
			}

			return flowstate.Noop(cronStateCtx), nil
		}

		if err := e.Do(flowstate.Commit(
			flowstate.Pause(cronStateCtx),
			flowstate.Delay(cronStateCtx, nextTimes[0].Sub(now)),
		)); err != nil {
			return nil, err
		}

		return flowstate.Noop(cronStateCtx), nil
	}))
	fr.SetFlow("task", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Commit(
			flowstate.End(stateCtx),
		), nil
	}))

	l, _ := NewTestLogger(t)
	e, err := flowstate.NewEngine(d, l)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, e.Shutdown(sCtx))
	}()

	dlr, err := flowstate.NewDelayer(e, l)
	require.NoError(t, err)
	defer func() {
		sCtx, sCtxCancel := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxCancel()

		require.NoError(t, dlr.Shutdown(sCtx))
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
	}, time.Second*20)
}
