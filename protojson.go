package flowstate

import (
	"encoding/base64"
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
	stateCtxs := commandStateCtxs(cmd0)
	datas := commandDatas(cmd0)

	jsonRootCmd, err := toJSONCommand(cmd0, stateCtxs, datas)
	if err != nil {
		return nil, err
	}
	jsonRootCmd.StateCtxs = stateCtxs
	jsonRootCmd.Datas = datas

	return json.Marshal(jsonRootCmd)
}

func toJSONCommand(cmd0 Command, stateCtxs stateCtxs, datas datas) (*jsonRootCommand, error) {
	jsonRootCmd := &jsonRootCommand{}

	switch cmd := cmd0.(type) {
	case *TransitCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}
		if cmd.To != "" {
			jsonCmd.To = &cmd.To
		}
		if cmd.Annotations != nil {
			jsonCmd.Annotations = &cmd.Annotations
		}

		jsonRootCmd.Transit = jsonCmd
	case *PauseCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.To != "" {
			jsonCmd.To = &cmd.To
		}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}

		jsonRootCmd.Pause = jsonCmd
	case *ResumeCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}
		if cmd.To != "" {
			jsonCmd.To = &cmd.To
		}

		jsonRootCmd.Resume = jsonCmd
	case *ParkCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}
		if cmd.Annotations != nil {
			jsonCmd.Annotations = &cmd.Annotations
		}

		jsonRootCmd.Park = jsonCmd
	case *ExecuteCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}

		jsonRootCmd.Execute = jsonCmd
	case *DelayCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
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
		if cmd.To != "" {
			jsonCmd.To = &cmd.To
		}

		jsonRootCmd.Delay = jsonCmd
	case *CommitCommand:
		commitJsonCmd := &jsonGenericCommand{}
		for _, cmd := range cmd.Commands {
			jsonCmd, err := toJSONCommand(cmd, stateCtxs, datas)
			if err != nil {
				return nil, fmt.Errorf("marshal commit command: %w", err)
			}

			commitJsonCmd.Commands = append(commitJsonCmd.Commands, jsonCmd)
		}

		jsonRootCmd.Commit = commitJsonCmd
	case *NoopCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}

		jsonRootCmd.Noop = jsonCmd
	case *StackCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.CarrierStateCtx != nil {
			jsonCmd.CarrierStateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.CarrierStateCtx),
			}
		}
		if cmd.StackedStateCtx != nil {
			jsonCmd.StackedStateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StackedStateCtx),
			}
		}
		if cmd.Annotation != "" {
			jsonCmd.Annotation = &cmd.Annotation
		}

		jsonRootCmd.Stack = jsonCmd
	case *UnstackCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.CarrierStateCtx != nil {
			jsonCmd.CarrierStateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.CarrierStateCtx),
			}
		}
		if cmd.UnstackStateCtx != nil {
			jsonCmd.UnstackStateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.UnstackStateCtx),
			}
		}
		if cmd.Annotation != "" {
			jsonCmd.Annotation = &cmd.Annotation
		}

		jsonRootCmd.Unstack = jsonCmd
	case *AttachDataCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}
		if cmd.Data != nil {
			jsonCmd.DataRef = &jsonDataRef{
				Idx: datas.strIdx(cmd.Data),
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
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}
		if cmd.Data != nil {
			jsonCmd.DataRef = &jsonDataRef{
				Idx: datas.strIdx(cmd.Data),
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
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}

		jsonRootCmd.GetStateByID = jsonCmd
	case *GetStateByLabelsCommand:
		jsonCmd := &jsonGenericCommand{}
		if cmd.Labels != nil {
			jsonCmd.Labels = &cmd.Labels
		}
		if cmd.StateCtx != nil {
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
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
			jsonCmd.StateRef = &jsonStateCtxRef{
				Idx: stateCtxs.strIdx(cmd.StateCtx),
			}
		}

		jsonRootCmd.CommitStateCtx = jsonCmd
	default:
		return nil, fmt.Errorf("unsupported command %T", cmd0)
	}

	return jsonRootCmd, nil
}

func UnmarshalJSONCommand(data []byte) (Command, error) {
	jsonRootCmd := &jsonRootCommand{}
	if err := json.Unmarshal(data, jsonRootCmd); err != nil {
		return nil, err
	}

	stateCtxs := stateCtxs(jsonRootCmd.StateCtxs)
	datas := datas(jsonRootCmd.Datas)

	return unmarshalJSONCommand(jsonRootCmd, stateCtxs, datas)
}

func unmarshalJSONCommand(jsonRootCmd *jsonRootCommand, stateCtxs stateCtxs, datas datas) (Command, error) {
	switch {
	case jsonRootCmd.Transit != nil:
		cmd := &TransitCommand{}

		if jsonRootCmd.Transit.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.Transit.StateRef.Idx)
		}
		if jsonRootCmd.Transit.To != nil {
			cmd.To = *jsonRootCmd.Transit.To
		}
		if jsonRootCmd.Transit.Annotations != nil {
			cmd.Annotations = *jsonRootCmd.Transit.Annotations
		}

		return cmd, nil
	case jsonRootCmd.Pause != nil:
		cmd := &PauseCommand{}

		if jsonRootCmd.Pause.To != nil {
			cmd.To = *jsonRootCmd.Pause.To
		}
		if jsonRootCmd.Pause.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.Pause.StateRef.Idx)
		}

		return cmd, nil
	case jsonRootCmd.Resume != nil:
		cmd := &ResumeCommand{}

		if jsonRootCmd.Resume.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.Resume.StateRef.Idx)
		}
		if jsonRootCmd.Resume.To != nil {
			cmd.To = *jsonRootCmd.Resume.To
		}

		return cmd, nil
	case jsonRootCmd.Park != nil:
		cmd := &ParkCommand{}

		if jsonRootCmd.Park.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.Park.StateRef.Idx)
		}
		if jsonRootCmd.Park.Annotations != nil {
			cmd.Annotations = *jsonRootCmd.Park.Annotations
		}

		return cmd, nil
	case jsonRootCmd.Execute != nil:
		cmd := &ExecuteCommand{}

		if jsonRootCmd.Execute.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.Execute.StateRef.Idx)
		}

		return cmd, nil
	case jsonRootCmd.Delay != nil:
		cmd := &DelayCommand{}

		if jsonRootCmd.Delay.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.Delay.StateRef.Idx)
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
		if jsonRootCmd.Delay.To != nil {
			cmd.To = *jsonRootCmd.Delay.To
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
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.Noop.StateRef.Idx)
		}

		return cmd, nil
	case jsonRootCmd.Stack != nil:
		cmd := &StackCommand{}

		if jsonRootCmd.Stack.CarrierStateRef != nil {
			cmd.CarrierStateCtx = stateCtxs.findStrIdx(jsonRootCmd.Stack.CarrierStateRef.Idx)
		}
		if jsonRootCmd.Stack.StackedStateRef != nil {
			cmd.StackedStateCtx = stateCtxs.findStrIdx(jsonRootCmd.Stack.StackedStateRef.Idx)
		}
		if jsonRootCmd.Stack.Annotation != nil {
			cmd.Annotation = *jsonRootCmd.Stack.Annotation
		}

		return cmd, nil
	case jsonRootCmd.Unstack != nil:
		cmd := &UnstackCommand{}

		if jsonRootCmd.Unstack.CarrierStateRef != nil {
			cmd.CarrierStateCtx = stateCtxs.findStrIdx(jsonRootCmd.Unstack.CarrierStateRef.Idx)
		}
		if jsonRootCmd.Unstack.UnstackStateRef != nil {
			cmd.UnstackStateCtx = stateCtxs.findStrIdx(jsonRootCmd.Unstack.UnstackStateRef.Idx)
		}
		if jsonRootCmd.Unstack.Annotation != nil {
			cmd.Annotation = *jsonRootCmd.Unstack.Annotation
		}

		return cmd, nil
	case jsonRootCmd.AttachData != nil:
		cmd := &AttachDataCommand{}

		if jsonRootCmd.AttachData.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.AttachData.StateRef.Idx)
		}
		if jsonRootCmd.AttachData.DataRef != nil {
			cmd.Data = datas.findStrIdx(jsonRootCmd.AttachData.DataRef.Idx)
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
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.GetData.StateRef.Idx)
		}
		if jsonRootCmd.GetData.DataRef != nil {
			cmd.Data = datas.findStrIdx(jsonRootCmd.GetData.DataRef.Idx)
		}
		if jsonRootCmd.GetData.Alias != nil {
			cmd.Alias = *jsonRootCmd.GetData.Alias
		}

		return cmd, nil
	case jsonRootCmd.GetStateByID != nil:
		cmd := &GetStateByIDCommand{}

		if jsonRootCmd.GetStateByID.StateRef != nil {
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.GetStateByID.StateRef.Idx)
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
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.GetStateByLabels.StateRef.Idx)
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
			cmd.StateCtx = stateCtxs.findStrIdx(jsonRootCmd.CommitStateCtx.StateRef.Idx)
		}

		return cmd, nil
	default:
		return nil, fmt.Errorf("unsupported command given")
	}
}

type jsonDataRef struct {
	Idx string `json:"idx,omitempty"`
}

type jsonStateCtxRef struct {
	Idx string `json:"idx,omitempty"`
}

type jsonRootCommand struct {
	StateCtxs []*StateCtx `json:"stateCtxs,omitempty"`
	Datas     []*Data     `json:"datas,omitempty"`

	Transit          *jsonGenericCommand          `json:"transit,omitempty"`
	Pause            *jsonGenericCommand          `json:"pause,omitempty"`
	Resume           *jsonGenericCommand          `json:"resume,omitempty"`
	Park             *jsonGenericCommand          `json:"park,omitempty"`
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
	StateRef   *jsonStateCtxRef `json:"stateRef,omitempty"`
	To         *FlowID          `json:"flowId,omitempty"`
	Transition *jsonTransition  `json:"transition,omitempty"`

	DelayingState *State  `json:"delayingState,omitempty"`
	ExecuteAtSec  *string `json:"executeAtSec,omitempty"`
	Commit        *bool   `json:"commit,omitempty"`

	Commands []*jsonRootCommand `json:"commands,omitempty"`

	StackedStateRef *jsonStateCtxRef `json:"stackedStateRef,omitempty"`
	CarrierStateRef *jsonStateCtxRef `json:"carrierStateRef,omitempty"`
	Annotation      *string          `json:"annotation,omitempty"`

	UnstackStateRef *jsonStateCtxRef `json:"unstackStateRef,omitempty"`

	DataRef *jsonDataRef `json:"dataRef,omitempty"`
	Alias   *string      `json:"alias,omitempty"`
	Store   *bool        `json:"store,omitempty"`

	ID  *StateID `json:"id,omitempty"`
	Rev *string  `json:"rev,omitempty"`

	Labels      *map[string]string `json:"labels,omitempty"`
	Annotations *map[string]string `json:"annotations,omitempty"`
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

type jsonData struct {
	ID     *string `json:"id,omitempty"`
	Rev    *string `json:"rev,omitempty"`
	Binary *bool   `json:"binary,omitempty"`
	B      *string `json:"b,omitempty"`
}

func (jsonData *jsonData) toData(d *Data) error {
	if jsonData.ID != nil {
		d.ID = DataID(*jsonData.ID)
	}
	if jsonData.Rev != nil && *jsonData.Rev != "" {
		rev, err := strconv.ParseInt(*jsonData.Rev, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse 'Rev rev = 1;' field: %w", err)
		}
		d.Rev = rev
	}
	if jsonData.Binary != nil {
		d.Binary = *jsonData.Binary
	}

	if jsonData.B != nil {
		if d.Binary {

			b, err := base64.StdEncoding.DecodeString(*jsonData.B)
			if err != nil {
				return err
			}
			d.B = b
		} else {
			d.B = []byte(*jsonData.B)
		}
	}
	return nil
}

func (jsonData *jsonData) fromData(d *Data) {
	if d.ID != "" {
		id := string(d.ID)
		jsonData.ID = &id
	}
	if d.Rev != 0 {
		rev := strconv.FormatInt(d.Rev, 10)
		jsonData.Rev = &rev
	}
	if d.Binary {
		jsonData.Binary = &d.Binary
	}

	if d.B != nil {
		var b string
		if d.Binary {
			b = base64.StdEncoding.EncodeToString(d.B)
		} else {
			b = string(d.B)
		}
		jsonData.B = &b
	}
	return
}

type jsonTransition struct {
	From        *string            `json:"from,omitempty"`
	To          *string            `json:"to,omitempty"`
	Annotations *map[string]string `json:"annotations,omitempty"`
}

func (jsonTs *jsonTransition) fromTransition(ts *Transition) {
	if ts.From != "" {
		from := string(ts.From)
		jsonTs.From = &from
	}
	if ts.To != "" {
		to := string(ts.To)
		jsonTs.To = &to
	}
	if ts.Annotations != nil {
		jsonTs.Annotations = &ts.Annotations
	}
}

func (jsonTs *jsonTransition) toTransition(ts *Transition) {
	if jsonTs.From != nil {
		ts.From = FlowID(*jsonTs.From)
	}
	if jsonTs.To != nil {
		ts.To = FlowID(*jsonTs.To)
	}
	if jsonTs.Annotations != nil {
		ts.Annotations = *jsonTs.Annotations
	}
}

type jsonState struct {
	ID          *string            `json:"id,omitempty"`
	Rev         *string            `json:"rev,omitempty"`
	Annotations *map[string]string `json:"annotations,omitempty"`
	Labels      *map[string]string `json:"labels,omitempty"`

	CommittedAtUnixMilli *string `json:"committedAtUnixMilli,omitempty"`

	Transition *jsonTransition `json:"transition,omitempty"`
}

func (jsonS *jsonState) fromState(s State) {
	if s.ID != "" {
		id := string(s.ID)
		jsonS.ID = &id
	}
	if s.Rev != 0 {
		rev := strconv.FormatInt(s.Rev, 10)
		jsonS.Rev = &rev
	}
	if s.Annotations != nil {
		jsonS.Annotations = &s.Annotations
	}
	if s.Labels != nil {
		jsonS.Labels = &s.Labels
	}
	if !s.CommittedAt.IsZero() {
		committedAt := strconv.FormatInt(s.CommittedAt.UnixMilli(), 10)
		jsonS.CommittedAtUnixMilli = &committedAt
	}

	jsonTs := &jsonTransition{}
	jsonTs.fromTransition(&s.Transition)
	jsonS.Transition = jsonTs
}

func (jsonS *jsonState) toState(s *State) error {
	if jsonS.ID != nil {
		s.ID = StateID(*jsonS.ID)
	}
	if jsonS.Rev != nil && *jsonS.Rev != "" {
		rev, err := strconv.ParseInt(*jsonS.Rev, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse 'Rev rev = 1;' field: %w", err)
		}
		s.Rev = rev
	}
	if jsonS.Annotations != nil {
		s.Annotations = *jsonS.Annotations
	}
	if jsonS.Labels != nil {
		s.Labels = *jsonS.Labels
	}
	if jsonS.CommittedAtUnixMilli != nil && *jsonS.CommittedAtUnixMilli != "" {
		unixMilli, err := strconv.ParseInt(*jsonS.CommittedAtUnixMilli, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse 'CommittedAtUnixMilli committed_at_unix_milli = 1;' field: %w", err)
		}
		s.CommittedAt = time.UnixMilli(unixMilli)
	}

	if jsonS.Transition != nil {
		jsonS.Transition.toTransition(&s.Transition)
	}

	return nil
}

type jsonStateCtx struct {
	Current     *jsonState        `json:"current,omitempty"`
	Committed   *jsonState        `json:"committed,omitempty"`
	Transitions []*jsonTransition `json:"transitions,omitempty"`
}

func (jsonSC *jsonStateCtx) fromStateCtx(s *StateCtx) {
	jsonSC.Current = &jsonState{}
	jsonSC.Current.fromState(s.Current)

	if s.Committed.Rev > 0 {
		jsonSC.Committed = &jsonState{}
		jsonSC.Committed.fromState(s.Committed)
	}

	for _, t := range s.Transitions {
		jsonT := &jsonTransition{}
		jsonT.fromTransition(&t)
		jsonSC.Transitions = append(jsonSC.Transitions, jsonT)
	}
}

func (jsonSC *jsonStateCtx) toStateCtx(s *StateCtx) error {
	if jsonSC.Current != nil {
		if err := jsonSC.Current.toState(&s.Current); err != nil {
			return fmt.Errorf("cannot convert current state: %w", err)
		}
	}
	if jsonSC.Committed != nil {
		if err := jsonSC.Committed.toState(&s.Committed); err != nil {
			return fmt.Errorf("cannot convert committed state: %w", err)
		}
	}

	for _, jsonT := range jsonSC.Transitions {
		t := Transition{}
		jsonT.toTransition(&t)
		s.Transitions = append(s.Transitions, t)
	}

	return nil
}

type jsonDelayedState struct {
	State        *jsonState `json:"state,omitempty"`
	Offset       *string    `json:"offset,omitempty"`
	ExecuteAtSec *string    `json:"executeAtSec,omitempty"`
}

func (jsonDS *jsonDelayedState) fromDelayedState(ds *DelayedState) {
	jsonDS.State = &jsonState{}
	jsonDS.State.fromState(ds.State)
	if ds.Offset != 0 {
		offset := strconv.FormatInt(ds.Offset, 10)
		jsonDS.Offset = &offset
	}
	if !ds.ExecuteAt.IsZero() {
		execAt := strconv.FormatInt(ds.ExecuteAt.Unix(), 10)
		jsonDS.ExecuteAtSec = &execAt
	}
}

func (jsonDS *jsonDelayedState) toDelayedState(ds *DelayedState) error {
	if jsonDS.State != nil {
		if err := jsonDS.State.toState(&ds.State); err != nil {
			return fmt.Errorf("cannot convert delayed state: %w", err)
		}
	}

	if jsonDS.Offset != nil {
		offset, err := strconv.ParseInt(*jsonDS.Offset, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse 'Offset offset = 1;' field: %w", err)
		}
		ds.Offset = offset
	}

	if jsonDS.ExecuteAtSec != nil && *jsonDS.ExecuteAtSec != "" {
		execAt, err := strconv.ParseInt(*jsonDS.ExecuteAtSec, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse 'ExecuteAtSec execute_at_sec = 2;' field: %w", err)
		}
		ds.ExecuteAt = time.Unix(execAt, 0)
	}

	return nil
}
