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
		Rev: 4,
		Annotations: map[string]string{
			"checksum/xxhash64": "11329650324634456925",
		},
		Blob: []byte(`50`),
	}
	actData := &flowstate.Data{}

	trkr := &Tracker{}

	mustSetFlow(fr, "foo", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(
			flowstate.GetData(stateCtx, `config_set`),
			flowstate.GetData(stateCtx, `data`),
		); err != nil {
			return nil, err
		}

		data := stateCtx.MustData(`data`)
		cfgs := stateCtx.MustData(`config_set`)

		var cfg fooConfigs
		if err := json.Unmarshal(cfgs.Blob, &cfg); err != nil {
			return nil, err
		}

		var num int
		if err := json.Unmarshal(data.Blob, &num); err != nil {
			return nil, err
		}

		num = cfg.Foo.A * num
		b, err := json.Marshal(num)
		if err != nil {
			return nil, err
		}
		data.Blob = append(data.Blob[:0], b...)

		stateCtx.SetData(`data`, data)

		if err := e.Do(
			flowstate.StoreData(stateCtx, `data`),
			flowstate.Transit(stateCtx, `bar`),
		); err != nil {
			return nil, err
		}

		return flowstate.Execute(stateCtx), nil
	}))
	mustSetFlow(fr, "bar", flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, e flowstate.Engine) (flowstate.Command, error) {
		Track(stateCtx, trkr)

		if err := e.Do(
			flowstate.GetData(stateCtx, `config_set`),
			flowstate.GetData(stateCtx, `data`),
		); err != nil {
			return nil, err
		}

		cfgs := stateCtx.MustData(`config_set`)
		data := stateCtx.MustData(`data`)

		var cfg barConfigs
		if err := json.Unmarshal(cfgs.Blob, &cfg); err != nil {
			return nil, err
		}

		var num int
		if err := json.Unmarshal(data.Blob, &num); err != nil {
			return nil, err
		}

		num = num / cfg.Bar.B
		b, err := json.Marshal(num)
		if err != nil {
			return nil, err
		}
		data.Blob = append(data.Blob[:0], b...)

		if err := e.Do(
			flowstate.StoreData(stateCtx, `data`),
			flowstate.Transit(stateCtx, `end`),
		); err != nil {
			return nil, err
		}

		data = stateCtx.MustData(`data`)
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
	stateCtx.SetData(`config_set`, &flowstate.Data{
		Blob: []byte(`
{
	"foo": {
		"a": 10
	},
	"bar": {
		"b": 20
	}
}
`),
	})

	stateCtx.SetData(`data`, &flowstate.Data{
		Blob: []byte(`100`),
	})

	require.NoError(t, e.Do(
		flowstate.StoreData(stateCtx, `config_set`),
		flowstate.StoreData(stateCtx, `data`),
	))

	require.NoError(t, e.Do(flowstate.Transit(stateCtx, `foo`)))
	require.NoError(t, e.Execute(stateCtx))

	trkr.WaitVisitedEqual(t, []string{`foo`, `bar`, `end`}, time.Second*2)
	require.Equal(t, expData, actData)
}
