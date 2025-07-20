package flowstate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func MarshalJSONState(s State) ([]byte, error) {
	return json.Marshal(s)
}

func UnmarshalJSONState(data []byte, s *State) error {
	return json.Unmarshal(data, &s)
}

func MarshalJSONTransition(ts Transition) ([]byte, error) {
	return json.Marshal(ts)
}

func UnmarshalJSONTransition(data []byte, ts *Transition) error {
	return json.Unmarshal(data, &ts)
}

func MarshalJSONStateCtx(s *StateCtx) ([]byte, error) {
	return json.Marshal(s)
}

func UnmarshalJSONStateCtx(data []byte, s *StateCtx) error {
	return json.Unmarshal(data, &s)
}

func MarshalJSONDelayedState(ds DelayedState) ([]byte, error) {
	return json.Marshal(ds)
}

func UnmarshalJSONDelayedState(data []byte, ds *DelayedState) error {
	return json.Unmarshal(data, &ds)
}

func MarshalJSONData(d *Data) ([]byte, error) {
	return json.Marshal(d)
}

func UnmarshalJSONData(data []byte, d *Data) error {
	return json.Unmarshal(data, &d)
}

func MarshalJSONCommand(cmd0 Command) ([]byte, error) {
	jsonRootCmd, err := toJSONCommand(cmd0)
	if err != nil {
		return nil, err
	}

	jsonRootCmd.StateCtxs = commandStateCtxs(cmd0)
	jsonRootCmd.Datas = commandDatas(cmd0)

	return json.Marshal(jsonRootCmd)
}

func toJSONCommand(cmd0 Command) (*jsonCommand, error) {
	jsonRootCmd := &jsonCommand{}

	switch cmd := cmd0.(type) {
	case *TransitCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.To != "" {
			jsonCmd.To = &cmd.To
		}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.Transit = jsonCmd
	case *PauseCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.To != "" {
			jsonCmd.To = &cmd.To
		}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.Pause = jsonCmd
	case *ResumeCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.Resume = jsonCmd
	case *EndCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.End = jsonCmd
	case *ExecuteCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.Execute = jsonCmd
	case *DelayCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}
		jsonCmd.DelayingState = &cmd.DelayingState
		if !cmd.ExecuteAt.IsZero() {
			sec := strconv.FormatInt(cmd.ExecuteAt.Unix(), 10)
			jsonCmd.ExecuteAtSec = &sec
		}
		if cmd.Commit {
			jsonCmd.Commit = &cmd.Commit
		}

		jsonRootCmd.Delay = jsonCmd
	case *CommitCommand:
		commitJsonCmd := &jsonGenericCommand{}
		for _, cmd := range cmd.Commands {
			jsonCmd, err := toJSONCommand(cmd)
			if err != nil {
				return nil, fmt.Errorf("marshal commit command: %w", err)
			}

			commitJsonCmd.Commands = append(commitJsonCmd.Commands, jsonCmd)
		}

		jsonRootCmd.Commit = commitJsonCmd
	case *NoopCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.Noop = jsonCmd
	case *StackCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.CarrierStateCtx != nil {
			jsonCmd.CarrierStateRef = &jsonStateRef{
				ID:  cmd.CarrierStateCtx.Current.ID,
				Rev: cmd.CarrierStateCtx.Current.Rev,
			}
		}
		if cmd.StackedStateCtx != nil {
			jsonCmd.StackedStateRef = &jsonStateRef{
				ID:  cmd.StackedStateCtx.Current.ID,
				Rev: cmd.StackedStateCtx.Current.Rev,
			}
		}
		if cmd.Annotation != "" {
			jsonCmd.Annotation = &cmd.Annotation
		}

		jsonRootCmd.Stack = jsonCmd
	case *UnstackCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.CarrierStateCtx != nil {
			jsonCmd.CarrierStateRef = &jsonStateRef{
				ID:  cmd.CarrierStateCtx.Current.ID,
				Rev: cmd.CarrierStateCtx.Current.Rev,
			}
		}
		if cmd.UnstackStateCtx != nil {
			jsonCmd.UnstackStateRef = &jsonStateRef{
				ID:  cmd.UnstackStateCtx.Current.ID,
				Rev: cmd.UnstackStateCtx.Current.Rev,
			}
		}
		if cmd.Annotation != "" {
			jsonCmd.Annotation = &cmd.Annotation
		}

		jsonRootCmd.Unstack = jsonCmd
	case *AttachDataCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}
		if cmd.Data != nil {
			jsonCmd.DataRef = &jsonDataRef{
				ID:  cmd.Data.ID,
				Rev: cmd.Data.Rev,
			}
		}
		if cmd.Alias != "" {
			jsonCmd.Alias = &cmd.Alias
		}
		if cmd.Store {
			jsonCmd.Store = &cmd.Store
		}

		jsonRootCmd.AttachData = jsonCmd
	case *GetDataCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}
		if cmd.Data != nil {
			jsonCmd.DataRef = &jsonDataRef{
				ID:  cmd.Data.ID,
				Rev: cmd.Data.Rev,
			}
		}
		if cmd.Alias != "" {
			jsonCmd.Alias = &cmd.Alias
		}

		jsonRootCmd.GetData = jsonCmd
	case *GetStateByIDCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.ID != "" {
			jsonCmd.ID = &cmd.ID
		}
		if cmd.Rev != 0 {
			rev := strconv.FormatInt(cmd.Rev, 10)
			jsonCmd.Rev = &rev
		}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.GetStateByID = jsonCmd
	case *GetStateByLabelsCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.Labels != nil {
			jsonCmd.Labels = &cmd.Labels
		}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.GetStateByLabels = jsonCmd
	case *GetStatesCommand:
		jsonCmd := &jsonGetStatesCommand{}
		if cmd.Labels != nil {
			jsonCmd.Labels = &cmd.Labels
		}
		if cmd.SinceRev != 0 {
			sinceRev := strconv.FormatInt(cmd.SinceRev, 10)
			jsonCmd.SinceRev = &sinceRev
		}
		if !cmd.SinceTime.IsZero() {
			usec := strconv.FormatInt(cmd.SinceTime.UnixMilli(), 10)
			jsonCmd.SinceTimeUsec = &usec
		}
		if cmd.LatestOnly {
			jsonCmd.LatestOnly = &cmd.LatestOnly
		}
		if cmd.Limit != 0 {
			limit := strconv.Itoa(cmd.Limit)
			jsonCmd.Limit = &limit
		}

		if cmd.Result != nil {
			res := &jsonGetStatesResult{}
			if cmd.Result.States != nil {
				res.States = &cmd.Result.States
			}
			if cmd.Result.More {
				res.More = &cmd.Result.More
			}
			jsonCmd.Result = res
		}

		jsonRootCmd.GetStates = jsonCmd
	case *GetDelayedStatesCommand:
		jsonCmd := &jsonGetDelayedStatesCommand{}
		if !cmd.Since.IsZero() {
			sec := strconv.FormatInt(cmd.Since.Unix(), 10)
			jsonCmd.SinceTimeSec = &sec
		}
		if !cmd.Until.IsZero() {
			sec := strconv.FormatInt(cmd.Until.Unix(), 10)
			jsonCmd.UntilTimeSec = &sec
		}
		if cmd.Offset != 0 {
			offset := strconv.FormatInt(cmd.Offset, 10)
			jsonCmd.Offset = &offset
		}
		if cmd.Limit != 0 {
			limit := strconv.Itoa(cmd.Limit)
			jsonCmd.Limit = &limit
		}

		if cmd.Result != nil {
			res := &jsonGetDelayedStatesResult{}
			if cmd.Result.States != nil {
				res.DelayedStates = &cmd.Result.States
			}
			if cmd.Result.More {
				res.More = &cmd.Result.More
			}
			jsonCmd.Result = res
		}

		jsonRootCmd.GetDelayedStates = jsonCmd
	case *CommitStateCtxCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateRef{
				ID:  cmd.StateCtx.Current.ID,
				Rev: cmd.StateCtx.Current.Rev,
			}
		}

		jsonRootCmd.CommitStateCtx = jsonCmd
	default:
		return nil, fmt.Errorf("unsupported command %T", cmd0)
	}

	return jsonRootCmd, nil
}

func UnmarshalJSONCommand(data []byte) (Command, error) {
	jsonRootCmd := &jsonCommand{}
	if err := json.Unmarshal(data, jsonRootCmd); err != nil {
		return nil, err
	}

	stateCtxs := stateCtxs(jsonRootCmd.StateCtxs)
	datas := datas(jsonRootCmd.Datas)

	return unmarshalJSONCommand(jsonRootCmd, stateCtxs, datas)

}

func unmarshalJSONCommand(jsonRootCmd *jsonCommand, stateCtxs stateCtxs, datas datas) (Command, error) {
	switch {
	case jsonRootCmd.Transit != nil:
		cmd := &TransitCommand{}

		if jsonRootCmd.Transit.To != nil {
			cmd.To = *jsonRootCmd.Transit.To
		}
		if jsonRootCmd.Transit.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}

		return cmd, nil
	case jsonRootCmd.Pause != nil:
		cmd := &PauseCommand{}

		if jsonRootCmd.Pause.To != nil {
			cmd.To = *jsonRootCmd.Pause.To
		}
		if jsonRootCmd.Pause.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.Pause.StateRef.ID, jsonRootCmd.Pause.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}

		return cmd, nil
	case jsonRootCmd.Resume != nil:
		cmd := &ResumeCommand{}

		if jsonRootCmd.Resume.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.Resume.StateRef.ID, jsonRootCmd.Resume.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}

		return cmd, nil
	case jsonRootCmd.End != nil:
		cmd := &EndCommand{}

		if jsonRootCmd.End.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.End.StateRef.ID, jsonRootCmd.End.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}

		return cmd, nil
	case jsonRootCmd.Execute != nil:
		cmd := &ExecuteCommand{}

		if jsonRootCmd.Execute.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.Execute.StateRef.ID, jsonRootCmd.Execute.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}

		return cmd, nil
	case jsonRootCmd.Delay != nil:
		cmd := &DelayCommand{}

		if jsonRootCmd.Delay.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.Delay.StateRef.ID, jsonRootCmd.Delay.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}
		if jsonRootCmd.Delay.DelayingState != nil {
			cmd.DelayingState = *jsonRootCmd.Delay.DelayingState
		}
		if jsonRootCmd.Delay.ExecuteAtSec != nil {
			execAt, err := strconv.ParseInt(*jsonRootCmd.Delay.ExecuteAtSec, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'ExecuteAtSec execute_at_sec = 3;' field: %w", err)
			}

			cmd.ExecuteAt = time.Unix(execAt, 0)
		}
		if jsonRootCmd.Delay.Commit != nil {
			cmd.Commit = *jsonRootCmd.Delay.Commit
		}

		return cmd, nil
	case jsonRootCmd.Commit != nil:
		cmd := &CommitCommand{}
		for _, jsonCmd := range jsonRootCmd.Commit.Commands {
			subCmd, err := unmarshalJSONCommand(jsonCmd, stateCtxs, datas)
			if err != nil {
				return nil, fmt.Errorf("unmarshal commit command: %w", err)
			}

			cmd.Commands = append(cmd.Commands, subCmd)
		}
		return cmd, nil
	case jsonRootCmd.Noop != nil:
		cmd := &NoopCommand{}

		if jsonRootCmd.Noop.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.Noop.StateRef.ID, jsonRootCmd.Noop.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}

		return cmd, nil
	case jsonRootCmd.Stack != nil:
		cmd := &StackCommand{}

		if jsonRootCmd.Stack.CarrierStateRef != nil {
			cmd.CarrierStateCtx = stateCtxs.find(jsonRootCmd.Stack.CarrierStateRef.ID, jsonRootCmd.Stack.CarrierStateRef.Rev)

			if cmd.CarrierStateCtx == nil {
				return nil, fmt.Errorf("cannot find CarrierStateCtx for 'CarrierStateRef carrier_state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Stack.CarrierStateRef.ID, jsonRootCmd.Stack.CarrierStateRef.Rev)
			}
		}
		if jsonRootCmd.Stack.StackedStateRef != nil {
			cmd.StackedStateCtx = stateCtxs.find(jsonRootCmd.Stack.StackedStateRef.ID, jsonRootCmd.Stack.StackedStateRef.Rev)

			if cmd.StackedStateCtx == nil {
				return nil, fmt.Errorf("cannot find CarrierStateCtx for 'CarrierStateRef carrier_state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Stack.StackedStateRef.ID, jsonRootCmd.Stack.StackedStateRef.Rev)
			}
		}
		if jsonRootCmd.Stack.Annotation != nil {
			cmd.Annotation = *jsonRootCmd.Stack.Annotation
		}

		return cmd, nil
	case jsonRootCmd.Unstack != nil:
		cmd := &UnstackCommand{}

		if jsonRootCmd.Unstack.CarrierStateRef != nil {
			cmd.CarrierStateCtx = stateCtxs.find(jsonRootCmd.Unstack.CarrierStateRef.ID, jsonRootCmd.Unstack.CarrierStateRef.Rev)

			if cmd.CarrierStateCtx == nil {
				return nil, fmt.Errorf("cannot find CarrierStateCtx for 'CarrierStateRef carrier_state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Unstack.CarrierStateRef.ID, jsonRootCmd.Unstack.CarrierStateRef.Rev)
			}
		}
		if jsonRootCmd.Unstack.UnstackStateRef != nil {
			cmd.UnstackStateCtx = stateCtxs.find(jsonRootCmd.Unstack.UnstackStateRef.ID, jsonRootCmd.Unstack.UnstackStateRef.Rev)

			if cmd.UnstackStateCtx == nil {
				return nil, fmt.Errorf("cannot find UnstackStateCtx for 'UnstackStateRef unstack_state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Unstack.UnstackStateRef.ID, jsonRootCmd.Unstack.UnstackStateRef.Rev)
			}
		}
		if jsonRootCmd.Unstack.Annotation != nil {
			cmd.Annotation = *jsonRootCmd.Unstack.Annotation
		}

		return cmd, nil
	case jsonRootCmd.AttachData != nil:
		cmd := &AttachDataCommand{}

		if jsonRootCmd.AttachData.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.AttachData.StateRef.ID, jsonRootCmd.AttachData.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.AttachData.StateRef.ID, jsonRootCmd.AttachData.StateRef.Rev)
			}
		}
		if jsonRootCmd.AttachData.DataRef != nil {
			cmd.Data = datas.find(jsonRootCmd.AttachData.DataRef.ID, jsonRootCmd.AttachData.DataRef.Rev)

			if cmd.Data == nil {
				return nil, fmt.Errorf("cannot find Data for 'DataRef data_ref = 1;' field with id %q and rev %d", jsonRootCmd.AttachData.DataRef.ID, jsonRootCmd.AttachData.DataRef.Rev)
			}
		}
		if jsonRootCmd.AttachData.Store != nil {
			cmd.Store = *jsonRootCmd.AttachData.Store
		}
		if jsonRootCmd.AttachData.Alias != nil {
			cmd.Alias = *jsonRootCmd.AttachData.Alias
		}

		return cmd, nil
	case jsonRootCmd.GetData != nil:
		cmd := &GetDataCommand{}

		if jsonRootCmd.GetData.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.GetData.StateRef.ID, jsonRootCmd.GetData.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.AttachData.StateRef.ID, jsonRootCmd.AttachData.StateRef.Rev)
			}
		}
		if jsonRootCmd.GetData.DataRef != nil {
			cmd.Data = datas.find(jsonRootCmd.GetData.DataRef.ID, jsonRootCmd.GetData.DataRef.Rev)

			if cmd.Data == nil {
				return nil, fmt.Errorf("cannot find Data for 'DataRef data_ref = 1;' field with id %q and rev %d", jsonRootCmd.AttachData.DataRef.ID, jsonRootCmd.AttachData.DataRef.Rev)
			}
		}
		if jsonRootCmd.GetData.Alias != nil {
			cmd.Alias = *jsonRootCmd.GetData.Alias
		}

		return cmd, nil
	case jsonRootCmd.GetStateByID != nil:
		cmd := &GetStateByIDCommand{}

		if jsonRootCmd.GetStateByID.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.GetStateByID.StateRef.ID, jsonRootCmd.GetStateByID.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.AttachData.StateRef.ID, jsonRootCmd.AttachData.StateRef.Rev)
			}
		}
		if jsonRootCmd.GetStateByID.ID != nil {
			cmd.ID = *jsonRootCmd.GetStateByID.ID
		}
		if jsonRootCmd.GetStateByID.Rev != nil {
			rev, err := strconv.ParseInt(*jsonRootCmd.GetStateByID.Rev, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse rev %q into integer", *jsonRootCmd.GetStateByID.Rev)
			}

			cmd.Rev = rev
		}

		return cmd, nil
	case jsonRootCmd.GetStateByLabels != nil:
		cmd := &GetStateByLabelsCommand{}

		if jsonRootCmd.GetStateByLabels.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.GetStateByLabels.StateRef.ID, jsonRootCmd.GetStateByLabels.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.AttachData.StateRef.ID, jsonRootCmd.AttachData.StateRef.Rev)
			}
		}
		if jsonRootCmd.GetStateByLabels.Labels != nil {
			cmd.Labels = *jsonRootCmd.GetStateByLabels.Labels
		}

		return cmd, nil
	case jsonRootCmd.GetStates != nil:
		cmd := &GetStatesCommand{}

		if jsonRootCmd.GetStates.Labels != nil {
			cmd.Labels = *jsonRootCmd.GetStates.Labels
		}
		if jsonRootCmd.GetStates.SinceRev != nil {
			rev, err := strconv.ParseInt(*jsonRootCmd.GetStates.SinceRev, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'SinceRev since_rev = 1;' field: %w", err)
			}

			cmd.SinceRev = rev
		}
		if jsonRootCmd.GetStates.SinceTimeUsec != nil {
			since, err := strconv.ParseInt(*jsonRootCmd.GetStates.SinceTimeUsec, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'SinceTimeUsec since_time_usec = 1;' field: %w", err)
			}

			cmd.SinceTime = time.UnixMilli(since)
		}
		if jsonRootCmd.GetStates.LatestOnly != nil {
			cmd.LatestOnly = *jsonRootCmd.GetStates.LatestOnly
		}
		if jsonRootCmd.GetStates.Limit != nil {
			limit, err := strconv.Atoi(*jsonRootCmd.GetStates.Limit)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'Limit limit = 4;' field: %w", err)
			}

			cmd.Limit = limit
		}

		if jsonRootCmd.GetStates.Result != nil {
			cmd.Result = &GetStatesResult{}
			if jsonRootCmd.GetStates.Result.States != nil {
				cmd.Result.States = *jsonRootCmd.GetStates.Result.States
			}
			if jsonRootCmd.GetStates.Result.More != nil {
				cmd.Result.More = *jsonRootCmd.GetStates.Result.More
			}
		}

		return cmd, nil
	case jsonRootCmd.GetDelayedStates != nil:
		cmd := &GetDelayedStatesCommand{}

		if jsonRootCmd.GetDelayedStates.SinceTimeSec != nil {
			since, err := strconv.ParseInt(*jsonRootCmd.GetDelayedStates.SinceTimeSec, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'SinceTimeSec since_time_sec = 1;' field: %w", err)
			}

			cmd.Since = time.Unix(since, 0)
		}
		if jsonRootCmd.GetDelayedStates.UntilTimeSec != nil {
			until, err := strconv.ParseInt(*jsonRootCmd.GetDelayedStates.UntilTimeSec, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'UntilTimeSec until_time_sec = 2;' field: %w", err)
			}

			cmd.Until = time.Unix(until, 0)
		}
		if jsonRootCmd.GetDelayedStates.Offset != nil {
			offset, err := strconv.ParseInt(*jsonRootCmd.GetDelayedStates.Offset, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'Offset offset = 3;' field: %w", err)
			}

			cmd.Offset = offset
		}
		if jsonRootCmd.GetDelayedStates.Limit != nil {
			limit, err := strconv.Atoi(*jsonRootCmd.GetDelayedStates.Limit)
			if err != nil {
				return nil, fmt.Errorf("cannot parse 'Limit limit = 4;' field: %w", err)
			}

			cmd.Limit = limit
		}

		if jsonRootCmd.GetDelayedStates.Result != nil {
			jsonRes := jsonRootCmd.GetDelayedStates.Result

			res := &GetDelayedStatesResult{}
			if jsonRes.DelayedStates != nil {
				res.States = *jsonRes.DelayedStates
			}
			if jsonRes.More != nil {
				res.More = *jsonRes.More
			}
			cmd.Result = res
		}

		return cmd, nil
	case jsonRootCmd.CommitStateCtx != nil:
		cmd := &CommitStateCtxCommand{}

		if jsonRootCmd.CommitStateCtx.StateRef != nil {
			cmd.StateCtx = stateCtxs.find(jsonRootCmd.CommitStateCtx.StateRef.ID, jsonRootCmd.CommitStateCtx.StateRef.Rev)

			if cmd.StateCtx == nil {
				return nil, fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", jsonRootCmd.Transit.StateRef.ID, jsonRootCmd.Transit.StateRef.Rev)
			}
		}

		return cmd, nil
	default:
		return nil, fmt.Errorf("unsupported command given")
	}
}

type jsonDataRef struct {
	ID  DataID `json:"id,omitempty"`
	Rev int64  `json:"rev,omitempty"`
}

type jsonStateRef struct {
	ID  StateID `json:"id,omitempty"`
	Rev int64   `json:"rev,omitempty"`
}

type jsonCommand struct {
	StateCtxs []*StateCtx `json:"state_ctxs,omitempty"`
	Datas     []*Data     `json:"datas,omitempty"`

	Transit          *jsonGenericCommand          `json:"transit,omitempty"`
	Pause            *jsonGenericCommand          `json:"pause,omitempty"`
	Resume           *jsonGenericCommand          `json:"resume,omitempty"`
	End              *jsonGenericCommand          `json:"end,omitempty"`
	Execute          *jsonGenericCommand          `json:"execute,omitempty"`
	Delay            *jsonGenericCommand          `json:"delay,omitempty"`
	Commit           *jsonGenericCommand          `json:"commit,omitempty"`
	Noop             *jsonGenericCommand          `json:"noop,omitempty"`
	Stack            *jsonGenericCommand          `json:"stack,omitempty"`
	Unstack          *jsonGenericCommand          `json:"unstack,omitempty"`
	AttachData       *jsonGenericCommand          `json:"attachData,omitempty"`
	GetData          *jsonGenericCommand          `json:"getData,omitempty"`
	GetStateByID     *jsonGenericCommand          `json:"getStateByID,omitempty"`
	GetStateByLabels *jsonGenericCommand          `json:"getStateByLabels,omitempty"`
	GetStates        *jsonGetStatesCommand        `json:"getStates,omitempty"`
	GetDelayedStates *jsonGetDelayedStatesCommand `json:"getDelayedStates,omitempty"`
	CommitStateCtx   *jsonGenericCommand          `json:"commitStateCtx,omitempty"`
}

type jsonGenericCommand struct {
	StateRef *jsonStateRef `json:"stateRef,omitempty"`
	To       *TransitionID `json:"flowId,omitempty"`

	DelayingState *State  `json:"delayingState,omitempty"`
	ExecuteAtSec  *string `json:"executeAtSec,omitempty"`
	Commit        *bool   `json:"commit,omitempty"`

	Commands []*jsonCommand `json:"commands,omitempty"`

	StackedStateRef *jsonStateRef `json:"stackedStateRef,omitempty"`
	CarrierStateRef *jsonStateRef `json:"carrierStateRef,omitempty"`
	Annotation      *string       `json:"annotation,omitempty"`

	UnstackStateRef *jsonStateRef `json:"unstackStateRef,omitempty"`

	DataRef *jsonDataRef `json:"dataRef,omitempty"`
	Alias   *string      `json:"alias,omitempty"`
	Store   *bool        `json:"store,omitempty"`

	ID  *StateID `json:"id,omitempty"`
	Rev *string  `json:"rev,omitempty"`

	Labels *map[string]string `json:"labels,omitempty"`
}

type jsonGetStatesCommand struct {
	SinceRev      *string              `json:"sinceRev,omitempty"`
	SinceTimeUsec *string              `json:"sinceTime_usec,omitempty"`
	Labels        *[]map[string]string `json:"labels,omitempty"`
	LatestOnly    *bool                `json:"latestOnly,omitempty"`
	Limit         *string              `json:"limit,omitempty"`

	Result *jsonGetStatesResult `json:"result,omitempty"`
}

type jsonGetStatesResult struct {
	States *[]State `json:"states,omitempty"`
	More   *bool    `json:"more,omitempty"`
}

type jsonGetDelayedStatesCommand struct {
	SinceTimeSec *string `json:"sinceTimeSec,omitempty"`
	UntilTimeSec *string `json:"untilTimeSec,omitempty"`
	Offset       *string `json:"offset,omitempty"`
	Limit        *string `json:"limit,omitempty"`

	Result *jsonGetDelayedStatesResult
}

type jsonGetDelayedStatesResult struct {
	DelayedStates *[]DelayedState `json:"delayedStates,omitempty"`
	More          *bool           `json:"more,omitempty"`
}
