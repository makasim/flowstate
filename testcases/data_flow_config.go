package testcases

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
)

func DataFlowConfig(t *testing.T, e flowstate.Engine, fr flowstate.FlowRegistry, d flowstate.Driver) {
	type fooConfig struct {
		A int `json:"a"`
	}
	type fooConfigs struct {
		Foo fooConfig `json:"foo"`
	}

	type barConfig struct {
		B int `json:"b"`
	}
	type barConfigs struct {
		Bar barConfig `json:"bar"`
	}

	expData := &flowstate.Data{
		ID:  `tid_data`,
		Rev: 4,
		B:   []byte(`50`),
	}
	actData := &flowstate.Data{}

	trkr := &Tracker{}

	mustSetFlow(fr, "foo", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		data := &flowstate.Data{}
		cfgs := &flowstate.Data{}
		if err := e.Do(
			flowstate.GetData(stateCtx, cfgs, `config_set`),
			flowstate.GetData(stateCtx, data, `data`),
		); err != nil {
			return nil, err
		}

		var cfg fooConfigs
		if err := json.Unmarshal(cfgs.B, &cfg); err != nil {
			return nil, err
		}

		var num int
		if err := json.Unmarshal(data.B, &num); err != nil {
			return nil, err
		}

		num = cfg.Foo.A * num
		b, err := json.Marshal(num)
		if err != nil {
			return nil, err
		}
		data.B = append(data.B[:0], b...)

		if err := e.Do(
			flowstate.AttachData(stateCtx, data, `data`),
			flowstate.Transit(stateCtx, `bar`),
		); err != nil {
			return nil, err
		}

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(fr, "bar", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		data := &flowstate.Data{}
		cfgs := &flowstate.Data{}
		if err := e.Do(
			flowstate.GetData(stateCtx, cfgs, `config_set`),
			flowstate.GetData(stateCtx, data, `data`),
		); err != nil {
			return nil, err
		}

		var cfg barConfigs
		if err := json.Unmarshal(cfgs.B, &cfg); err != nil {
			return nil, err
		}

		var num int
		if err := json.Unmarshal(data.B, &num); err != nil {
			return nil, err
		}

		num = num / cfg.Bar.B
		b, err := json.Marshal(num)
		if err != nil {
			return nil, err
		}
		data.B = append(data.B[:0], b...)

		if err := e.Do(
			flowstate.AttachData(stateCtx, data, `data`),
			flowstate.Transit(stateCtx, `end`),
		); err != nil {
			return nil, err
		}

		data.CopyTo(actData)

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(fr, "end", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)
		return flowstate.Park(stateCtx), nil
	}))

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: "TID",
		},
	}

	configSet := &flowstate.Data{
		ID: `first_config_set`,
		B: []byte(`
{
	"foo": {
		"a": 10
	},
	"bar": {
		"b": 20
	}
}
`),
	}

	data := &flowstate.Data{
		ID: `tid_data`,
		B:  []byte(`100`),
	}

	require.NoError(t, e.Do(
		flowstate.AttachData(stateCtx, configSet, `config_set`),
		flowstate.AttachData(stateCtx, data, `data`),
	))

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `foo`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitVisitedEqual(t, []string{`foo`, `bar`, `end`}, time.Second*2)
	require.Equal(t, expData, actData)
}
