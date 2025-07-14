package flowstate

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/VictoriaMetrics/easyproto"
)

var mp = &easyproto.MarshalerPool{}

func MarshalState(s State, dst []byte) []byte {
	m := mp.Get()
	defer mp.Put(m)

	marshalState(s, m.MessageMarshaler())
	return m.Marshal(dst)
}

//	message State {
//	 string id = 1;
//	 int64 rev = 2;
//	 map<string, string> annotations = 3;
//	 map<string, string> labels = 4;
//	 int64 committed_at_unix_milli = 5;
//	 Transition transition = 6;
//	}
//
// from https://github.com/makasim/flowstatesrv/blob/073eb0c28cbc438104ab700cb89b55c4401d3c19/proto/flowstate/v1/state.proto#L28
func marshalState(s State, mm *easyproto.MessageMarshaler) {
	if s.ID != "" {
		mm.AppendString(1, string(s.ID))
	}
	if s.Rev != 0 {
		mm.AppendInt64(2, s.Rev)
	}
	if len(s.Annotations) > 0 {
		marshalStringMap(s.Annotations, 3, mm)
	}
	if len(s.Labels) > 0 {
		marshalStringMap(s.Labels, 4, mm)
	}
	if s.CommittedAtUnixMilli != 0 {
		mm.AppendInt64(5, s.CommittedAtUnixMilli)
	}

	marshalTransition(s.Transition, mm.AppendMessage(6))
}

func UnmarshalState(src []byte, s *State) (err error) {
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}
		switch fc.FieldNum {
		case 1:
			id, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string id = 1;' field")
			}
			s.ID = StateID(strings.Clone(id))
		case 2:
			rev, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 rev = 2;' field")
			}
			s.Rev = rev
		case 3:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'map<string, string> annotations = 3;' field")
			}

			if s.Annotations == nil {
				s.Annotations = make(map[string]string)
			}

			if err := unmarshalStringMapItem(data, s.Annotations); err != nil {
				return fmt.Errorf("cannot read 'map<string, string> annotations = 3;' field: %w", err)
			}
		case 4:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'map<string, string> labels = 4;' field")
			}

			if s.Labels == nil {
				s.Labels = make(map[string]string)
			}

			if err := unmarshalStringMapItem(data, s.Labels); err != nil {
				return fmt.Errorf("cannot read 'map<string, string> labels = 4;' field: %w", err)
			}
		case 5:
			timestamp, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 committed_at_unix_milli = 5;' field")
			}
			s.CommittedAtUnixMilli = timestamp
		case 6:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'Transition transition = 6;' field")
			}

			if err := UnmarshalTransition(data, &s.Transition); err != nil {
				return fmt.Errorf("cannot unmarshal 'Transition transition = 6;' field: %w", err)
			}
		}
	}
	return nil
}

func MarshalStateCtx(stateCtx *StateCtx, dst []byte) []byte {
	m := mp.Get()
	defer mp.Put(m)

	marshalStateCtx(stateCtx, m.MessageMarshaler())
	return m.Marshal(dst)
}

//	message StateContext {
//	 State committed = 1;
//	 State current = 2;
//	 repeated Transition transitions = 3;
//	}
//
// from https://github.com/makasim/flowstatesrv/blob/073eb0c28cbc438104ab700cb89b55c4401d3c19/proto/flowstate/v1/state.proto#L22
func marshalStateCtx(stateCtx *StateCtx, mm *easyproto.MessageMarshaler) {
	marshalState(stateCtx.Committed, mm.AppendMessage(1))
	marshalState(stateCtx.Current, mm.AppendMessage(2))

	for _, s := range stateCtx.Transitions {
		marshalTransition(s, mm.AppendMessage(3))
	}
}

func UnmarshalStateCtx(src []byte, stateCtx *StateCtx) (err error) {
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}
		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'State committed = 1;' field")
			}

			if err := UnmarshalState(data, &stateCtx.Committed); err != nil {
				return fmt.Errorf("cannot unmarshal 'State committed = 1;' field: %w", err)
			}
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'State current = 2;' field")
			}

			if err := UnmarshalState(data, &stateCtx.Current); err != nil {
				return fmt.Errorf("cannot unmarshal 'State current = 2;' field: %w", err)
			}
		case 3:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'repeated Transition transitions = 3;' field")
			}

			ts := Transition{}
			if err := UnmarshalTransition(data, &ts); err != nil {
				return fmt.Errorf("cannot unmarshal 'repeated Transition transitions = 3;' field: %w", err)
			}

			stateCtx.Transitions = append(stateCtx.Transitions, ts)
		}
	}
	return nil
}

func MarshalDelayedState(ds DelayedState, dst []byte) []byte {
	m := mp.Get()
	defer mp.Put(m)

	marshalDelayedState(ds, m.MessageMarshaler())
	return m.Marshal(dst)
}

//	message DelayedState {
//	 State state = 1;
//	 int64 offset = 2;
//	 int64 execute_at_sec = 3;
//	}
//
// from https://github.com/makasim/flowstatesrv/blob/eeac827d8c3a935bb49393bb552091839b9cd438/proto/flowstate/v1/state.proto#L37
func marshalDelayedState(ds DelayedState, mm *easyproto.MessageMarshaler) {
	marshalState(ds.State, mm.AppendMessage(1))

	if ds.Offset != 0 {
		mm.AppendInt64(2, ds.Offset)
	}
	if !ds.ExecuteAt.IsZero() {
		mm.AppendInt64(3, ds.ExecuteAt.Unix())
	}
}

func UnmarshalDelayedState(src []byte, ds *DelayedState) (err error) {
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}
		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'State state = 1;' field")
			}

			if err := UnmarshalState(data, &ds.State); err != nil {
				return fmt.Errorf("cannot unmarshal 'State state = 1;' field: %w", err)
			}
		case 2:
			offset, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 offset = 2;' field")
			}
			ds.Offset = offset
		case 3:
			executeAtSec, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 execute_at_sec = 3;' field")
			}

			if executeAtSec != 0 {
				ds.ExecuteAt = time.Unix(executeAtSec, 0)
			}
		}
	}
	return nil
}

func MarshalTransition(ts Transition, dst []byte) []byte {
	m := mp.Get()
	defer mp.Put(m)

	marshalTransition(ts, m.MessageMarshaler())

	return m.Marshal(dst)
}

//	message Transition {
//	 string from = 1;
//	 string to = 2;
//	 map<string, string> annotations = 3;
//	}
//
// from https://github.com/makasim/flowstatesrv/blob/main/proto/flowstate/v1/state.proto#L37
func marshalTransition(ts Transition, mm *easyproto.MessageMarshaler) {
	if ts.From != "" {
		mm.AppendString(1, string(ts.From))
	}
	if ts.To != "" {
		mm.AppendString(2, string(ts.To))
	}

	if len(ts.Annotations) > 0 {
		marshalStringMap(ts.Annotations, 3, mm)
	}
}

func UnmarshalTransition(src []byte, ts *Transition) (err error) {
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}
		switch fc.FieldNum {
		case 1:
			from, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string from = 1;' field")
			}
			ts.From = TransitionID(strings.Clone(from))
		case 2:
			to, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string to = 2;' field")
			}
			ts.To = TransitionID(strings.Clone(to))
		case 3:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'map<string, string> annotations = 3;' field")
			}

			if ts.Annotations == nil {
				ts.Annotations = make(map[string]string)
			}

			if err := unmarshalStringMapItem(data, ts.Annotations); err != nil {
				return fmt.Errorf("cannot read 'map<string, string> annotations = 3;' field: %w", err)
			}
		}
	}
	return nil
}

func MarshalData(d *Data, dst []byte) []byte {
	m := mp.Get()
	defer mp.Put(m)

	marshalData(d, m.MessageMarshaler())

	return m.Marshal(dst)
}

//	message Data {
//	 string id = 1;
//	 int64 rev = 2;
//	 bool binary = 3;
//	 string b = 4;
//	}
//
// https://github.com/makasim/flowstatesrv/blob/eeac827d8c3a935bb49393bb552091839b9cd438/proto/flowstate/v1/state.proto#L5C1-L10C2
func marshalData(d *Data, mm *easyproto.MessageMarshaler) {
	if d.ID != "" {
		mm.AppendString(1, string(d.ID))
	}
	if d.Rev != 0 {
		mm.AppendInt64(2, d.Rev)
	}
	if d.Binary {
		mm.AppendBool(3, d.Binary)
	}
	if len(d.B) > 0 {
		dst := d.B
		if d.Binary {
			dst = make([]byte, base64.StdEncoding.EncodedLen(len(d.B)))
			base64.StdEncoding.Encode(dst, d.B)
		}

		mm.AppendString(4, string(dst))
	}
}

func UnmarshalData(src []byte, d *Data) (err error) {
	var b string

	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}
		switch fc.FieldNum {
		case 1:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string id = 1;' field")
			}
			d.ID = DataID(v)
		case 2:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 rev = 2;' field")
			}
			d.Rev = v
		case 3:
			v, ok := fc.Bool()
			if !ok {
				return fmt.Errorf("cannot read 'bool binary = 3;' field")
			}
			d.Binary = v
		case 4:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string b = 4;' field")
			}

			b = v
		}
	}

	if len(b) == 0 {
		return nil
	}

	if d.Binary {
		b, err := base64.StdEncoding.AppendDecode(d.B[:0], []byte(b))
		if err != nil {
			return fmt.Errorf("cannot decode 'string b = 4;' field from base64: %w", err)
		}
		d.B = b
	} else {
		d.B = []byte(strings.Clone(b))
	}

	return nil
}

func MarshalCommand(cmd Command, dst []byte) []byte {
	m := mp.Get()
	defer mp.Put(m)

	marshalCommand(cmd, false, m.MessageMarshaler())

	return m.Marshal(dst)
}

//	message Command {
//	 repeated StateCtx state_ctxs = 1;
//	 repeated Data datas = 2;
//
//	 TransitCommand transit = 3;
//	 PauseCommand pause = 4;
//	 ResumeCommand resume = 5;
//	 EndCommand end = 6;
//	 ExecuteCommand execute = 7;
//	 DelayCommand delay = 8;
//	 CommitCommand commit = 9;
//	 NoopCommand noop = 10;
//	 SerializeCommand serialize = 11;
//	 DeserializeCommand deserialize = 12;
//	 StoreDataCommand store_data = 13;
//	 GetDataCommand get_data = 14;
//	 ReferenceDataCommand reference_data = 15;
//	 DereferenceDataCommand dereference_data = 16;
//	 GetStateByIDCommand get_state_by_id = 17;
//	 GetStateByLabelsCommand get_state_by_labels = 18;
//	 GetStatesCommand get_states = 19;
//	 GetDelayedStatesCommand get_delayed_states = 20;
//	 CommitStateCtxCommand commit_state = 21;
//	}
func marshalCommand(cmd0 Command, sub bool, mm *easyproto.MessageMarshaler) {
	if !sub {
		if stateCtxs := commandStateCtxs(cmd0); len(stateCtxs) > 0 {
			for _, stateCtx := range stateCtxs {
				marshalStateCtx(stateCtx, mm.AppendMessage(1))
			}
		}
		if datas := commandDatas(cmd0); len(datas) > 0 {
			for _, data := range datas {
				marshalData(data, mm.AppendMessage(2))
			}
		}
	}

	switch cmd := cmd0.(type) {
	case *TransitCommand:
		//	message TransitCommand {
		//	 StateRef state_ref = 1;
		//	 string flow_id = 2;
		//	}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}

			if cmd.To != "" {
				mm.AppendString(2, string(cmd.To))
			}
		}(mm.AppendMessage(3))
	case *PauseCommand:
		//	message PauseCommand {
		//	 StateRef state_ref = 1;
		//	 string flow_id = 2;
		//	}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}

			if cmd.To != "" {
				mm.AppendString(2, string(cmd.To))
			}
		}(mm.AppendMessage(4))
	case *ResumeCommand:
		//	message ResumeCommand {
		//	 StateRef state_ref = 1;
		//	}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(5))
	case *EndCommand:
		//	message EndCommand {
		//	 StateRef state_ref = 1;
		//	}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(6))
	case *ExecuteCommand:
		//	message ExecuteCommand {
		//	 StateRef state_ref = 1;
		//	}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(7))
	case *DelayCommand:
		//	message DelayCommand {
		//	 StateRef state_ref = 1;
		//	 State delaying_state = 2;
		//	 int64 execute_at_sec = 3;
		//	 bool commit = 4;
		//	}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
			marshalState(cmd.DelayingState, mm.AppendMessage(2))
			if !cmd.ExecuteAt.IsZero() {
				mm.AppendInt64(3, cmd.ExecuteAt.Unix())
			}
			if cmd.Commit {
				mm.AppendBool(4, true)
			}
		}(mm.AppendMessage(8))
	case *CommitCommand:
		// message CommitCommand {
		//  repeated Command commands = 1;
		// }
		func(mm *easyproto.MessageMarshaler) {
			for _, subCmd := range cmd.Commands {
				marshalCommand(subCmd, true, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(9))
	case *NoopCommand:
		//	message NoopCommand {
		//	 StateRef state_ref = 1;
		//	}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(10))
	case *SerializeCommand:
		// message SerializeCommand {
		//  StateRef serializable_state_ref = 1;
		//  StateRef state_ref = 2;
		//  string annotation = 3;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if cmd.SerializableStateCtx != nil {
				marshalStateRef(cmd.SerializableStateCtx.Current, mm.AppendMessage(1))
			}
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(2))
			}
			if cmd.Annotation != "" {
				mm.AppendString(3, cmd.Annotation)
			}
		}(mm.AppendMessage(11))
	case *DeserializeCommand:
		// message DeserializeCommand {
		//  StateRef deserialized_state_ref = 1;
		//  StateRef state_ref = 2;
		//  string annotation = 3;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if cmd.DeserializedStateCtx != nil {
				marshalStateRef(cmd.DeserializedStateCtx.Current, mm.AppendMessage(1))
			}
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(2))
			}
			if cmd.Annotation != "" {
				mm.AppendString(3, cmd.Annotation)
			}
		}(mm.AppendMessage(12))
	case *StoreDataCommand:
		// message StoreDataCommand {
		//  DataRef data_ref = 1;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if cmd.Data != nil {
				marshalDataRef(cmd.Data, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(13))
	case *GetDataCommand:
		// message GetDataCommand {
		//  DataRef data_ref = 1;
		//}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.Data != nil {
				marshalDataRef(cmd.Data, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(14))
	case *ReferenceDataCommand:
		// message ReferenceDataCommand {
		//  StateRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string annotation = 3;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
			if cmd.Data != nil {
				marshalDataRef(cmd.Data, mm.AppendMessage(2))
			}
			if cmd.Annotation != "" {
				mm.AppendString(3, cmd.Annotation)
			}
		}(mm.AppendMessage(15))
	case *DereferenceDataCommand:
		// message DereferenceDataCommand {
		//  StateRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string annotation = 3;
		//}
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
			if cmd.Data != nil {
				marshalDataRef(cmd.Data, mm.AppendMessage(2))
			}
			if cmd.Annotation != "" {
				mm.AppendString(3, cmd.Annotation)
			}
		}(mm.AppendMessage(16))
	case *GetStateByIDCommand:
		// message GetStateByIDCommand {
		//  string id = 1;
		//  int64 rev = 2;
		//  StateRef state_ref = 3;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if cmd.ID != "" {
				mm.AppendString(1, string(cmd.ID))
			}
			if cmd.Rev != 0 {
				mm.AppendInt64(2, cmd.Rev)
			}
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(3))
			}
		}(mm.AppendMessage(17))
	case *GetStateByLabelsCommand:
		// message GetStateByLabelsCommand {
		//  map<string, string> labels = 1;
		//  StateRef state_ref = 2;
		//}
		func(mm *easyproto.MessageMarshaler) {
			if len(cmd.Labels) > 0 {
				marshalStringMap(cmd.Labels, 1, mm)
			}
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(2))
			}
		}(mm.AppendMessage(18))
	case *GetStatesCommand:
		// message GetStatesCommand {
		//  message Labels {
		//    map<string, string> labels = 1;
		//  }
		//
		//  message Result {
		//    repeated State states = 1;
		//    bool more = 2;
		//  }
		//
		//  int64 since_rev = 1;
		//  int64 since_time_usec = 2; // unix milliseconds
		//  repeated Labels labels = 3;
		//  bool latest_only = 4;
		//  int64 limit = 5;
		//
		//  Result result = 6;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if cmd.SinceRev != 0 {
				mm.AppendInt64(1, cmd.SinceRev)
			}
			if !cmd.SinceTime.IsZero() {
				mm.AppendInt64(2, cmd.SinceTime.UnixMilli())
			}
			if len(cmd.Labels) > 0 {
				for _, labels := range cmd.Labels {
					lMM := mm.AppendMessage(3)
					marshalStringMap(labels, 1, lMM)
				}
			}
			if cmd.LatestOnly {
				mm.AppendBool(4, true)
			}
			if cmd.Limit != 0 {
				mm.AppendInt64(5, int64(cmd.Limit))
			}

			if cmd.Result != nil {
				resultMM := mm.AppendMessage(6)
				for _, state := range cmd.Result.States {
					marshalState(state, resultMM.AppendMessage(1))
				}
				if cmd.Result.More {
					resultMM.AppendBool(2, true)
				}
			}
		}(mm.AppendMessage(19))
	case *GetDelayedStatesCommand:
		// message GetDelayedStatesCommand {
		//  message Result {
		//    repeated DelayedState delayed_states = 1;
		//    bool more = 2;
		//  }
		//
		//  int64 since_time_sec = 1;
		//  int64 until_time_sec = 2;
		//  int64 offset = 3;
		//  int64 limit = 4;
		//
		//  Result result = 5;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if !cmd.Since.IsZero() {
				mm.AppendInt64(1, cmd.Since.Unix())
			}
			if !cmd.Until.IsZero() {
				mm.AppendInt64(2, cmd.Until.Unix())
			}
			if cmd.Offset != 0 {
				mm.AppendInt64(3, cmd.Offset)
			}
			if cmd.Limit != 0 {
				mm.AppendInt64(4, int64(cmd.Limit))
			}

			if cmd.Result != nil {
				resultMM := mm.AppendMessage(5)
				for _, delayedState := range cmd.Result.States {
					marshalDelayedState(delayedState, resultMM.AppendMessage(1))
				}
				if cmd.Result.More {
					resultMM.AppendBool(2, true)
				}
			}
		}(mm.AppendMessage(20))
	case *CommitStateCtxCommand:
		// message CommitStateCtxCommand {
		//   StateRef state_ref = 1;
		// }
		func(mm *easyproto.MessageMarshaler) {
			if cmd.StateCtx != nil {
				marshalStateRef(cmd.StateCtx.Current, mm.AppendMessage(1))
			}
		}(mm.AppendMessage(21))
	default:
		// TODO: return an error on unknown command ??
		// TODO: return an error if no command specified ??
	}
}

func UnmarshalCommand(src0 []byte) (Command, error) {
	var stateCtxs stateCtxs
	var datas datas
	var err error

	// unmarshal stateCtxs and datas first
	var fc easyproto.FieldContext
	src := src0
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return nil, fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'repeated StateCtx state_ctxs = 1;' field")
			}

			stateCtx := &StateCtx{}
			if err := UnmarshalStateCtx(data, stateCtx); err != nil {
				return nil, fmt.Errorf("cannot read 'repeated StateCtx state_ctxs = 1;' field: %w", err)
			}

			stateCtxs = append(stateCtxs, stateCtx)
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'repeated Data datas = 2;' field")
			}

			d := &Data{}
			if err := UnmarshalData(data, d); err != nil {
				return nil, fmt.Errorf("cannot read 'repeated Data datas = 2;' field: %w", err)
			}

			datas = append(datas, d)
		}
	}

	return unmarshalCommand(src0, stateCtxs, datas)
}

func unmarshalCommand(src []byte, stateCtxs stateCtxs, datas datas) (Command, error) {
	var err error

	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return nil, fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 3: //	 TransitCommand transit = 3;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'TransitCommand transit = 3;' field")
			}

			cmd := &TransitCommand{}
			if err := unmarshalTransitCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'TransitCommand transit = 3;' field: %w", err)
			}
			return cmd, nil
		case 4: //	 PauseCommand pause = 4;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'PauseCommand pause = 4;' field")
			}

			cmd := &PauseCommand{}
			if err := unmarshalPauseCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'PauseCommand pause = 4;' field: %w", err)
			}
			return cmd, nil
		case 5: //	 ResumeCommand resume = 5;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'ResumeCommand resume = 5;' field")
			}

			cmd := &ResumeCommand{}
			if err := unmarshalResumeCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'ResumeCommand resume = 5;' field: %w", err)
			}
			return cmd, nil
		case 6: // EndCommand end = 6;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'EndCommand end = 6;' field")
			}

			cmd := &EndCommand{}
			if err := unmarshalEndCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'EndCommand end = 6;' field: %w", err)
			}
			return cmd, nil
		case 7: // ExecuteCommand execute = 7;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'ExecuteCommand execute = 7;' field")
			}

			cmd := &ExecuteCommand{}
			if err := unmarshalExecuteCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'ExecuteCommand execute = 7;' field: %w", err)
			}

			return cmd, nil
		case 8: // DelayCommand delay = 8;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'DelayCommand delay = 8;' field")
			}

			cmd := &DelayCommand{}
			if err := unmarshalDelayCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'DelayCommand delay = 8;' field: %w", err)
			}

			return cmd, nil
		case 9: // CommitCommand commit = 9;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'CommitCommand commit = 9;' field")
			}

			cmd := &CommitCommand{}
			if err := unmarshalCommitCommand(data, cmd, stateCtxs, datas); err != nil {
				return nil, fmt.Errorf("cannot read 'CommitCommand commit = 9;' field: %w", err)
			}

			return cmd, nil
		case 10: // NoopCommand noop = 10;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'NoopCommand noop = 10;' field")
			}

			cmd := &NoopCommand{}
			if err := unmarshalNoopCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'NoopCommand noop = 10;' field: %w", err)
			}

			return cmd, nil
		case 11: // SerializeCommand serialize = 11;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'SerializeCommand serialize = 11;' field")
			}

			cmd := &SerializeCommand{}
			if err := unmarshalSerializeCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'SerializeCommand serialize = 11;' field: %w", err)
			}

			return cmd, nil
		case 12: // DeserializeCommand deserialize = 12;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'DeserializeCommand deserialize = 12;' field")
			}

			cmd := &DeserializeCommand{}
			if err := unmarshalDeserializeCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'DeserializeCommand deserialize = 12;' field: %w", err)
			}

			return cmd, nil
		case 13: // StoreDataCommand store_data = 13;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'StoreDataCommand store_data = 13;' field")
			}

			cmd := &StoreDataCommand{}
			if err := unmarshalStoreDataCommand(data, cmd, datas); err != nil {
				return nil, fmt.Errorf("cannot read 'StoreDataCommand store_data = 13;' field: %w", err)
			}

			return cmd, nil
		case 14: // GetDataCommand get_data = 14;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'GetDataCommand get_data = 14;' field")
			}

			cmd := &GetDataCommand{}
			if err := unmarshalGetDataCommand(data, cmd, datas); err != nil {
				return nil, fmt.Errorf("cannot read 'GetDataCommand get_data = 14;' field: %w", err)
			}

			return cmd, nil
		case 15: // ReferenceDataCommand reference_data = 15;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'ReferenceDataCommand reference_data = 15;' field")
			}

			cmd := &ReferenceDataCommand{}
			if err := unmarshalReferenceDataCommand(data, cmd, stateCtxs, datas); err != nil {
				return nil, fmt.Errorf("cannot read 'ReferenceDataCommand reference_data = 15;' field: %w", err)
			}

			return cmd, nil
		case 16: // DereferenceDataCommand dereference_data = 16;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'DereferenceDataCommand dereference_data = 16;' field")
			}

			cmd := &DereferenceDataCommand{}
			if err := unmarshalDereferenceDataCommand(data, cmd, stateCtxs, datas); err != nil {
				return nil, fmt.Errorf("cannot read 'DereferenceDataCommand dereference_data = 16;' field: %w", err)
			}

			return cmd, nil
		case 17: // GetStateByIDCommand get_state_by_id = 17;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'GetStateByIDCommand get_state_by_id = 17;' field")
			}

			cmd := &GetStateByIDCommand{}
			if err := unmarshalGetStateByIDCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'GetStateByIDCommand get_state_by_id = 17;' field: %w", err)
			}
			return cmd, nil
		case 18: // GetStateByLabelsCommand get_state_by_labels = 18;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'GetStateByLabelsCommand get_state_by_labels = 18;' field")
			}

			cmd := &GetStateByLabelsCommand{}
			if err := unmarshalGetStateByLabelsCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'GetStateByLabelsCommand get_state_by_labels = 18;' field: %w", err)
			}
			return cmd, nil
		case 19: // GetStatesCommand get_states = 19;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'GetStatesCommand get_states = 19;' field")
			}

			cmd := &GetStatesCommand{}
			if err := unmarshalGetStatesCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'GetStatesCommand get_states = 19;' field: %w", err)
			}
			return cmd, nil
		case 20: // GetDelayedStatesCommand get_delayed_states = 20;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'GetDelayedStatesCommand get_delayed_states = 20;' field")
			}

			cmd := &GetDelayedStatesCommand{}
			if err := unmarshalGetDelayedStatesCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'GetDelayedStatesCommand get_delayed_states = 20;' field: %w", err)
			}
			return cmd, nil
		case 21: // CommitStateCtxCommand commit_state = 21;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read ' CommitStateCtxCommand commit_state = 21;' field")
			}

			cmd := &CommitStateCtxCommand{}
			if err := unmarshalCommitStateCtxCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read ' CommitStateCtxCommand commit_state = 21;' field: %w", err)
			}
			return cmd, nil
		}
	}

	return nil, nil
}

func unmarshalTransitCommand(src []byte, cmd *TransitCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 2:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string flow_id = 2;' field")
			}

			cmd.To = TransitionID(strings.Clone(v))
		}
	}

	return nil
}

func unmarshalPauseCommand(src []byte, cmd *PauseCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 2:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string flow_id = 2;' field")
			}

			cmd.To = TransitionID(strings.Clone(v))
		}
	}

	return nil
}

func unmarshalResumeCommand(src []byte, cmd *ResumeCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalEndCommand(src []byte, cmd *EndCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalExecuteCommand(src []byte, cmd *ExecuteCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalDelayCommand(src []byte, cmd *DelayCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'State delaying_state = 2;' field")
			}

			if err := UnmarshalState(data, &cmd.DelayingState); err != nil {
				return fmt.Errorf("cannot read 'State delaying_state = 2;' field: %w", err)
			}
		case 3:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 execute_at_sec = 3;' field")
			}

			cmd.ExecuteAt = time.Unix(v, 0)
		case 4:
			v, ok := fc.Bool()
			if !ok {
				return fmt.Errorf("cannot read 'bool commit = 4;' field")
			}

			cmd.Commit = v
		}
	}

	return nil
}

func unmarshalCommitCommand(src []byte, cmd *CommitCommand, stateCtxs stateCtxs, datas datas) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message CommitCommand {
		//  repeated Command commands = 1;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			sumCmd, err := unmarshalCommand(data, stateCtxs, datas)
			if err != nil {
				return fmt.Errorf("cannot read 'repeated Command commands = 1;' field: %w", err)
			}

			cmd.Commands = append(cmd.Commands, sumCmd)
		}
	}

	return nil
}

func unmarshalNoopCommand(src []byte, cmd *NoopCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalSerializeCommand(src []byte, cmd *SerializeCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message SerializeCommand {
		//  StateRef serializable_state_ref = 1;
		//  StateRef state_ref = 2;
		//  string annotation = 3;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef serializable_state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef serializable_state_ref = 1;' field: %w", err)
			}

			cmd.SerializableStateCtx = stateCtxs.find(id, rev)
			if cmd.SerializableStateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef serializable_state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 2;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 2;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 2;' field with id %q and rev %d", id, rev)
			}
		case 3:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string annotation = 3;' field")
			}
			cmd.Annotation = strings.Clone(v)
		}
	}

	return nil
}

func unmarshalDeserializeCommand(src []byte, cmd *DeserializeCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message DeserializeCommand {
		//  StateRef deserialized_state_ref = 1;
		//  StateRef state_ref = 2;
		//  string annotation = 3;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef deserialized_state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef deserialized_state_ref = 1;' field: %w", err)
			}

			cmd.DeserializedStateCtx = stateCtxs.find(id, rev)
			if cmd.DeserializedStateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef deserialized_state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 2;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 2;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 2;' field with id %q and rev %d", id, rev)
			}
		case 3:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string annotation = 3;' field")
			}
			cmd.Annotation = strings.Clone(v)
		}
	}

	return nil
}

func unmarshalStoreDataCommand(src []byte, cmd *StoreDataCommand, datas datas) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message StoreDataCommand {
		//  DataRef data_ref = 1;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field")
			}

			id, rev, err := unmarshalDataRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field: %w", err)
			}

			cmd.Data = datas.find(id, rev)
			if cmd.Data == nil {
				return fmt.Errorf("cannot find StateCtx for 'DataRef data_ref = 1;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalGetDataCommand(src []byte, cmd *GetDataCommand, datas datas) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message GetDataCommand {
		//  DataRef data_ref = 1;
		//}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field")
			}

			id, rev, err := unmarshalDataRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field: %w", err)
			}

			cmd.Data = datas.find(id, rev)
			if cmd.Data == nil {
				return fmt.Errorf("cannot find StateCtx for 'DataRef data_ref = 1;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalReferenceDataCommand(src []byte, cmd *ReferenceDataCommand, stateCtxs stateCtxs, datas datas) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message ReferenceDataCommand {
		//  StateRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string annotation = 3;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field")
			}

			id, rev, err := unmarshalDataRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field: %w", err)
			}

			cmd.Data = datas.find(id, rev)
			if cmd.Data == nil {
				return fmt.Errorf("cannot find StateCtx for 'DataRef data_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 3:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string annotation = 3;' field")
			}

			cmd.Annotation = strings.Clone(v)
		}
	}

	return nil
}

func unmarshalDereferenceDataCommand(src []byte, cmd *DereferenceDataCommand, stateCtxs stateCtxs, datas datas) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message DereferenceDataCommand {
		//  StateRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string annotation = 3;
		//}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field")
			}

			id, rev, err := unmarshalDataRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field: %w", err)
			}

			cmd.Data = datas.find(id, rev)
			if cmd.Data == nil {
				return fmt.Errorf("cannot find StateCtx for 'DataRef data_ref = 1;' field with id %q and rev %d", id, rev)
			}
		case 3:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string annotation = 3;' field")
			}

			cmd.Annotation = strings.Clone(v)
		}
	}

	return nil
}

func unmarshalGetStateByIDCommand(src []byte, cmd *GetStateByIDCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message GetStateByIDCommand {
		//  string id = 1;
		//  int64 rev = 2;
		//  StateRef state_ref = 3;
		// }

		switch fc.FieldNum {
		case 1:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string id = 1;' field")
			}
			cmd.ID = StateID(strings.Clone(v))
		case 2:
			value, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 rev = 2;' field")
			}
			cmd.Rev = value
		case 3:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 3;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 3;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 3;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalGetStateByLabelsCommand(src []byte, cmd *GetStateByLabelsCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message GetStateByLabelsCommand {
		//  map<string, string> labels = 1;
		//  StateRef state_ref = 2;
		//}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'map<string, string> labels = 1;' field")
			}
			if cmd.Labels == nil {
				cmd.Labels = make(map[string]string)
			}

			if err := unmarshalStringMapItem(data, cmd.Labels); err != nil {
				return fmt.Errorf("cannot read 'map<string, string> labels = 1;' field: %w", err)
			}
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 2;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 2;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 2;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

func unmarshalGetStatesCommand(src []byte, cmd *GetStatesCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message GetStatesCommand {
		//  message Labels {
		//    map<string, string> labels = 1;
		//  }
		//
		//  message Result {
		//    repeated State states = 1;
		//    bool more = 2;
		//  }
		//
		//  int64 since_rev = 1;
		//  int64 since_time_usec = 2; // unix milliseconds
		//  repeated Labels labels = 3;
		//  bool latest_only = 4;
		//  int64 limit = 5;
		//
		//  Result result = 6;
		// }

		switch fc.FieldNum {
		case 1:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 since_rev = 1;' field")
			}
			cmd.SinceRev = v
		case 2:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 since_time_usec = 2;' field")
			}
			cmd.SinceTime = time.UnixMilli(v)
		case 3:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'repeated Labels labels = 3;' field")
			}

			labelsFC := easyproto.FieldContext{}
			labels := make(map[string]string)
			for len(data) > 0 {
				data, err = labelsFC.NextField(data)
				if err != nil {
					return fmt.Errorf("cannot read next field")
				}
				switch labelsFC.FieldNum {
				case 1:
					labelsData, ok := labelsFC.MessageData()
					if !ok {
						return fmt.Errorf("cannot read 'Result result = 6;' field")
					}
					if err := unmarshalStringMapItem(labelsData, labels); err != nil {
						return fmt.Errorf("cannot read 'repeated Labels labels = 3;' field: %w", err)
					}
				}
			}

			cmd.Labels = append(cmd.Labels, labels)
		case 4:
			v, ok := fc.Bool()
			if !ok {
				return fmt.Errorf("cannot read 'bool latest_only = 4;' field")
			}
			cmd.LatestOnly = v
		case 5:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 limit = 5;' field")
			}
			cmd.Limit = int(v)
		case 6:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'Result result = 6;' field")
			}

			resultFC := easyproto.FieldContext{}
			result := &GetStatesResult{}
			for len(data) > 0 {
				data, err = resultFC.NextField(data)
				if err != nil {
					return fmt.Errorf("cannot read next field")
				}
				switch resultFC.FieldNum {
				case 1:
					stateData, ok := resultFC.MessageData()
					if !ok {
						return fmt.Errorf("cannot read 'Result result = 6;' field")
					}

					state := State{}
					if err := UnmarshalState(stateData, &state); err != nil {
						return fmt.Errorf("cannot read 'Result result = 6;' field: %w", err)
					}

					result.States = append(result.States, state)
				case 2:
					v, ok := resultFC.Bool()
					if !ok {
						return fmt.Errorf("cannot read 'bool more = 2;' field")
					}
					result.More = v
				}
			}

			cmd.Result = result
		}
	}

	return nil
}

func unmarshalGetDelayedStatesCommand(src []byte, cmd *GetDelayedStatesCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message GetDelayedStatesCommand {
		//  message Result {
		//    repeated DelayedState delayed_states = 1;
		//    bool more = 2;
		//  }
		//
		//  int64 since_time_sec = 1;
		//  int64 until_time_sec = 2;
		//  int64 offset = 3;
		//  int64 limit = 4;
		//
		//  Result result = 5;
		// }

		switch fc.FieldNum {
		case 1:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 since_time_sec = 1;' field")
			}
			cmd.Since = time.Unix(v, 0)
		case 2:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 until_time_sec = 2;' field")
			}
			cmd.Until = time.Unix(v, 0)
		case 3:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 offset = 3;' field")
			}
			cmd.Offset = v
		case 4:
			v, ok := fc.Int64()
			if !ok {
				return fmt.Errorf("cannot read 'int64 limit = 4;' field")
			}
			cmd.Limit = int(v)
		case 5:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'Result result = 5;' field")
			}

			resultFC := easyproto.FieldContext{}
			result := &GetDelayedStatesResult{}
			for len(data) > 0 {
				data, err = resultFC.NextField(data)
				if err != nil {
					return fmt.Errorf("cannot read next field")
				}
				switch resultFC.FieldNum {
				case 1:
					stateData, ok := resultFC.MessageData()
					if !ok {
						return fmt.Errorf("cannot read 'Result result = 5;' field")
					}

					ds := DelayedState{}
					if err := UnmarshalDelayedState(stateData, &ds); err != nil {
						return fmt.Errorf("cannot read 'Result result = 6;' field: %w", err)
					}

					result.States = append(result.States, ds)
				case 2:
					v, ok := resultFC.Bool()
					if !ok {
						return fmt.Errorf("cannot read 'bool more = 2;' field")
					}
					result.More = v
				}
			}

			cmd.Result = result
		}
	}

	return nil
}

func unmarshalCommitStateCtxCommand(src []byte, cmd *CommitStateCtxCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field")
			}

			id, rev, err := unmarshalStateRef(data)
			if err != nil {
				return fmt.Errorf("cannot read 'StateRef state_ref = 1;' field: %w", err)
			}

			cmd.StateCtx = stateCtxs.find(id, rev)
			if cmd.StateCtx == nil {
				return fmt.Errorf("cannot find StateCtx for 'StateRef state_ref = 1;' field with id %q and rev %d", id, rev)
			}
		}
	}

	return nil
}

//	message StateRef {
//	 string id = 1;
//	 int64 rev = 2;
//	}
//
// from https://github.com/makasim/flowstatesrv/blob/eeac827d8c3a935bb49393bb552091839b9cd438/proto/flowstate/v1/state.proto#L17
func marshalStateRef(state State, mm *easyproto.MessageMarshaler) {
	if state.ID != "" {
		mm.AppendString(1, string(state.ID))
	}
	if state.Rev != 0 {
		mm.AppendInt64(2, state.Rev)
	}
}

func unmarshalStateRef(src []byte) (StateID, int64, error) {
	var fc easyproto.FieldContext

	var id StateID
	var rev int64
	var err error
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return "", 0, fmt.Errorf("cannot read next field: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			v, ok := fc.String()
			if !ok {
				return "", 0, fmt.Errorf("cannot read 'string id = 1;' field")
			}
			id = StateID(strings.Clone(v))
		case 2:
			value, ok := fc.Int64()
			if !ok {
				return "", 0, fmt.Errorf("cannot read 'int64 rev = 2;' field")
			}
			rev = value
		}
	}

	return id, rev, nil
}

//	message DataRef {
//	 string id = 1;
//	 int64 rev = 2;
//	}
//
// from https://github.com/makasim/flowstatesrv/blob/eeac827d8c3a935bb49393bb552091839b9cd438/proto/flowstate/v1/state.proto#L12
func marshalDataRef(data *Data, mm *easyproto.MessageMarshaler) {
	if data.ID != "" {
		mm.AppendString(1, string(data.ID))
	}
	if data.Rev != 0 {
		mm.AppendInt64(2, data.Rev)
	}
}

func unmarshalDataRef(src []byte) (DataID, int64, error) {
	var fc easyproto.FieldContext

	var id DataID
	var rev int64
	var err error
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return "", 0, fmt.Errorf("cannot read next field: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			v, ok := fc.String()
			if !ok {
				return "", 0, fmt.Errorf("cannot read 'string id = 1;' field")
			}
			id = DataID(strings.Clone(v))
		case 2:
			value, ok := fc.Int64()
			if !ok {
				return "", 0, fmt.Errorf("cannot read 'int64 rev = 2;' field")
			}
			rev = value
		}
	}

	return id, rev, nil
}

func commandStateCtxs(cmd0 Command) []*StateCtx {
	stateCtxs := make([]*StateCtx, 0)

	switch cmd := cmd0.(type) {
	case *TransitCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *PauseCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *ResumeCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *EndCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *ExecuteCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *DelayCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *NoopCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *SerializeCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
		if cmd.SerializableStateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.SerializableStateCtx)
		}
	case *DeserializeCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
		if cmd.DeserializedStateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.DeserializedStateCtx)
		}
	case *CommitCommand:
		for _, subCmd := range cmd.Commands {
			stateCtxs = append(stateCtxs, commandStateCtxs(subCmd)...)
		}
	case *ReferenceDataCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *DereferenceDataCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *GetStateByIDCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *GetStateByLabelsCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *CommitStateCtxCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	default:
		return nil
	}

	slices.CompactFunc(stateCtxs, func(l, r *StateCtx) bool {
		return l.Current.ID == r.Current.ID && l.Current.Rev == r.Current.Rev
	})

	return stateCtxs
}

func commandDatas(cmd0 Command) []*Data {
	datas := make([]*Data, 0)

	switch cmd := cmd0.(type) {
	case *StoreDataCommand:
		if cmd.Data != nil {
			datas = append(datas, cmd.Data)
		}
	case *GetDataCommand:
		if cmd.Data != nil {
			datas = append(datas, cmd.Data)
		}
	case *ReferenceDataCommand:
		if cmd.Data != nil {
			datas = append(datas, cmd.Data)
		}
	case *DereferenceDataCommand:
		if cmd.Data != nil {
			datas = append(datas, cmd.Data)
		}
	case *CommitCommand:
		for _, subCmd := range cmd.Commands {
			datas = append(datas, commandDatas(subCmd)...)
		}
	default:
		return nil
	}

	slices.CompactFunc(datas, func(l, r *Data) bool {
		return l.ID == r.ID && l.Rev == r.Rev
	})

	return datas
}

func unmarshalStringMapItem(src []byte, m map[string]string) (err error) {
	var fc easyproto.FieldContext

	var key, value string
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			key0, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read map key")
			}
			if key0 == "" {
				return fmt.Errorf("map key cannot be empty")
			}

			key = strings.Clone(key0)
		case 2:
			value0, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read map value")
			}
			value = strings.Clone(value0)
		}
	}

	m[key] = value
	return nil
}

func marshalStringMap(m map[string]string, fieldNum uint32, mm *easyproto.MessageMarshaler) {
	for k, v := range m {
		itemMM := mm.AppendMessage(fieldNum)
		itemMM.AppendString(1, k)
		if v != "" {
			itemMM.AppendString(2, v)
		}
	}
}

type stateCtxs []*StateCtx

func (s stateCtxs) find(id StateID, rev int64) *StateCtx {
	for _, stateCtx := range s {
		if stateCtx.Current.ID == id && stateCtx.Current.Rev == rev {
			return stateCtx
		}
	}
	return nil
}

type datas []*Data

func (d datas) find(id DataID, rev int64) *Data {
	for _, data := range d {
		if data.ID == id && data.Rev == rev {
			return data
		}
	}
	return nil
}
