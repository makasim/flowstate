package flowstate

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strconv"
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
	if !s.CommittedAt.IsZero() {
		mm.AppendInt64(5, s.CommittedAt.UnixMilli())
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
			s.CommittedAt = time.UnixMilli(timestamp)
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
func marshalTransition(ts Transition, mm *easyproto.MessageMarshaler) {
	if ts.From != "" {
		mm.AppendString(1, string(ts.From))
	}
	if ts.To != "" {
		mm.AppendString(2, string(ts.To))
	}

	if ts.Annotations != nil {
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
			ts.From = FlowID(strings.Clone(from))
		case 2:
			to, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string to = 2;' field")
			}
			ts.To = FlowID(strings.Clone(to))
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

	mm := m.MessageMarshaler()

	stateCtxs := commandStateCtxs(cmd)
	if len(stateCtxs) > 0 {
		for _, stateCtx := range stateCtxs {
			marshalStateCtx(stateCtx, mm.AppendMessage(1))
		}
	}
	datas := commandDatas(cmd)
	if len(datas) > 0 {
		for _, data := range datas {
			marshalData(data, mm.AppendMessage(2))
		}
	}

	marshalCommand(cmd, stateCtxs, datas, mm)

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
//	 StackCommand stack = 11;
//	 UnstackCommand unstack = 12;
//	 AttachDataCommand attach_data = 13;
//	 GetDataCommand get_data = 14;
//	 GetStateByIDCommand get_state_by_id = 17;
//	 GetStateByLabelsCommand get_state_by_labels = 18;
//	 GetStatesCommand get_states = 19;
//	 GetDelayedStatesCommand get_delayed_states = 20;
//	 CommitStateCtxCommand commit_state = 21;
//	}
func marshalCommand(cmd0 Command, stateCtxs stateCtxs, datas datas, mm *easyproto.MessageMarshaler) {
	switch cmd := cmd0.(type) {
	case *TransitCommand:
		//	message TransitCommand {
		//	 StateCtxRef state_ref = 1;
		//	 string to = 2;
		//	 Transition transition = 3;
		//	}
		cmdMM := mm.AppendMessage(3)

		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}

		if cmd.To != "" {
			cmdMM.AppendString(2, string(cmd.To))
		}
		if cmd.Transition.To != "" {
			marshalTransition(cmd.Transition, cmdMM.AppendMessage(3))
		}
	case *PauseCommand:
		//	message PauseCommand {
		//	 StateCtxRef state_ref = 1;
		//	 string to = 2;
		//	}
		cmdMM := mm.AppendMessage(4)

		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}

		if cmd.To != "" {
			cmdMM.AppendString(2, string(cmd.To))
		}
	case *ResumeCommand:
		//	message ResumeCommand {
		//	 StateCtxRef state_ref = 1;
		//	 string to = 2;
		//	}
		cmdMM := mm.AppendMessage(5)

		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
		if cmd.To != "" {
			cmdMM.AppendString(2, string(cmd.To))
		}
	case *EndCommand:
		//	message EndCommand {
		//	 StateCtxRef state_ref = 1;
		//	 string to = 2;
		//	}
		cmdMM := mm.AppendMessage(6)

		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
		if cmd.To != "" {
			cmdMM.AppendString(2, string(cmd.To))
		}
	case *ExecuteCommand:
		//	message ExecuteCommand {
		//	 StateCtxRef state_ref = 1;
		//	}
		cmdMM := mm.AppendMessage(7)

		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
	case *DelayCommand:
		//	message DelayCommand {
		//	 StateCtxRef state_ref = 1;
		//	 State delaying_state = 2;
		//	 int64 execute_at_sec = 3;
		//	 bool commit = 4;
		//	 string to = 5;
		//	}
		cmdMM := mm.AppendMessage(8)
		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
		marshalState(cmd.DelayingState, cmdMM.AppendMessage(2))
		if !cmd.ExecuteAt.IsZero() {
			cmdMM.AppendInt64(3, cmd.ExecuteAt.Unix())
		}
		if cmd.Commit {
			cmdMM.AppendBool(4, true)
		}
		if cmd.To != "" {
			cmdMM.AppendString(5, string(cmd.To))
		}
	case *CommitCommand:
		// message CommitCommand {
		//  repeated Command commands = 1;
		// }
		cmdMM := mm.AppendMessage(9)
		for _, subCmd := range cmd.Commands {
			marshalCommand(subCmd, stateCtxs, datas, cmdMM.AppendMessage(1))
		}
	case *NoopCommand:
		//	message NoopCommand {
		//	 StateCtxRef state_ref = 1;
		//	}
		cmdMM := mm.AppendMessage(10)
		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
	case *StackCommand:
		// message StackCommand {
		//  StateCtxRef carrier_state_ref = 1;
		//  StateCtxRef stack_state_ref = 2;
		//  string annotation = 3;
		// }
		cmdMM := mm.AppendMessage(11)
		if cmd.CarrierStateCtx != nil {
			marshalStateCtxRef(cmd.CarrierStateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
		if cmd.StackedStateCtx != nil {
			marshalStateCtxRef(cmd.StackedStateCtx, stateCtxs, cmdMM.AppendMessage(2))
		}
		if cmd.Annotation != "" {
			cmdMM.AppendString(3, cmd.Annotation)
		}
	case *UnstackCommand:
		// message UnstackCommand {
		//  StateCtxRef carrier_state_ref = 1;
		//  StateCtxRef unstack_state_ref = 2;
		//  string annotation = 3;
		// }
		cmdMM := mm.AppendMessage(12)
		if cmd.CarrierStateCtx != nil {
			marshalStateCtxRef(cmd.CarrierStateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
		if cmd.UnstackStateCtx != nil {
			marshalStateCtxRef(cmd.UnstackStateCtx, stateCtxs, cmdMM.AppendMessage(2))
		}
		if cmd.Annotation != "" {
			cmdMM.AppendString(3, cmd.Annotation)
		}
	case *AttachDataCommand:
		// message AttachDataCommand {
		//  StateCtxRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string alias = 3;
		//  bool store = 4;
		// }
		cmdMM := mm.AppendMessage(13)
		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
		if cmd.Data != nil {
			marshalDataRef(cmd.Data, datas, cmdMM.AppendMessage(2))
		}
		if cmd.Alias != "" {
			cmdMM.AppendString(3, cmd.Alias)
		}
		if cmd.Store {
			cmdMM.AppendBool(4, true)
		}
	case *GetDataCommand:
		// message GetDataCommand {
		//  StateCtxRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string alias = 3;
		//}
		cmdMM := mm.AppendMessage(14)
		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
		if cmd.Data != nil {
			marshalDataRef(cmd.Data, datas, cmdMM.AppendMessage(2))
		}
		if cmd.Alias != "" {
			cmdMM.AppendString(3, cmd.Alias)
		}
	case *GetStateByIDCommand:
		// message GetStateByIDCommand {
		//  string id = 1;
		//  int64 rev = 2;
		//  StateCtxRef state_ref = 3;
		// }
		cmdMM := mm.AppendMessage(17)
		if cmd.ID != "" {
			cmdMM.AppendString(1, string(cmd.ID))
		}
		if cmd.Rev != 0 {
			cmdMM.AppendInt64(2, cmd.Rev)
		}
		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(3))
		}
	case *GetStateByLabelsCommand:
		// message GetStateByLabelsCommand {
		//  map<string, string> labels = 1;
		//  StateCtxRef state_ref = 2;
		//}
		cmdMM := mm.AppendMessage(18)
		if len(cmd.Labels) > 0 {
			marshalStringMap(cmd.Labels, 1, cmdMM)
		}
		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(2))
		}
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
		cmdMM := mm.AppendMessage(19)
		if cmd.SinceRev != 0 {
			cmdMM.AppendInt64(1, cmd.SinceRev)
		}
		if !cmd.SinceTime.IsZero() {
			cmdMM.AppendInt64(2, cmd.SinceTime.UnixMilli())
		}
		if len(cmd.Labels) > 0 {
			for _, labels := range cmd.Labels {
				lMM := cmdMM.AppendMessage(3)
				marshalStringMap(labels, 1, lMM)
			}
		}
		if cmd.LatestOnly {
			cmdMM.AppendBool(4, true)
		}
		if cmd.Limit != 0 {
			cmdMM.AppendInt64(5, int64(cmd.Limit))
		}

		if cmd.Result != nil {
			resultMM := cmdMM.AppendMessage(6)
			for _, state := range cmd.Result.States {
				marshalState(state, resultMM.AppendMessage(1))
			}
			if cmd.Result.More {
				resultMM.AppendBool(2, true)
			}
		}
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
		cmdMM := mm.AppendMessage(20)
		if !cmd.Since.IsZero() {
			cmdMM.AppendInt64(1, cmd.Since.Unix())
		}
		if !cmd.Until.IsZero() {
			cmdMM.AppendInt64(2, cmd.Until.Unix())
		}
		if cmd.Offset != 0 {
			cmdMM.AppendInt64(3, cmd.Offset)
		}
		if cmd.Limit != 0 {
			cmdMM.AppendInt64(4, int64(cmd.Limit))
		}

		if cmd.Result != nil {
			resultMM := cmdMM.AppendMessage(5)
			for _, delayedState := range cmd.Result.States {
				marshalDelayedState(delayedState, resultMM.AppendMessage(1))
			}
			if cmd.Result.More {
				resultMM.AppendBool(2, true)
			}
		}
	case *CommitStateCtxCommand:
		// message CommitStateCtxCommand {
		//   StateCtxRef state_ref = 1;
		// }
		cmdMM := mm.AppendMessage(21)
		if cmd.StateCtx != nil {
			marshalStateCtxRef(cmd.StateCtx, stateCtxs, cmdMM.AppendMessage(1))
		}
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
		case 11: // StackCommand stack = 11;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'StackCommand stack = 11;' field")
			}

			cmd := &StackCommand{}
			if err := unmarshalStackCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'StackCommand stack = 11;' field: %w", err)
			}

			return cmd, nil
		case 12: // UnstackCommand unstack = 12;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'UnstackCommand unstack = 12;' field")
			}

			cmd := &UnstackCommand{}
			if err := unmarshalUnstackCommand(data, cmd, stateCtxs); err != nil {
				return nil, fmt.Errorf("cannot read 'UnstackCommand unstack = 12;' field: %w", err)
			}

			return cmd, nil
		case 13: // AttachDataCommand attach_data = 13;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'AttachDataCommand attach_data = 13;' field")
			}

			cmd := &AttachDataCommand{}
			if err := unmarshalAttachDataCommand(data, cmd, stateCtxs, datas); err != nil {
				return nil, fmt.Errorf("cannot read 'AttachDataCommand attach_data = 13;' field: %w", err)
			}

			return cmd, nil
		case 14: // GetDataCommand get_data = 14;
			data, ok := fc.MessageData()
			if !ok {
				return nil, fmt.Errorf("cannot read 'GetDataCommand get_data = 14;' field")
			}

			cmd := &GetDataCommand{}
			if err := unmarshalGetDataCommand(data, cmd, stateCtxs, datas); err != nil {
				return nil, fmt.Errorf("cannot read 'GetDataCommand get_data = 14;' field: %w", err)
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		case 2:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string to = 2;' field")
			}

			cmd.To = FlowID(strings.Clone(v))
		case 3:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'Transition transition = 3;' field")
			}

			if err := UnmarshalTransition(data, &cmd.Transition); err != nil {
				return fmt.Errorf("cannot read 'Transition transition = 3;' field: %w", err)
			}
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		case 2:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string to = 2;' field")
			}

			cmd.To = FlowID(strings.Clone(v))
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		case 2:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string to = 2;' field")
			}

			cmd.To = FlowID(strings.Clone(v))
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		case 2:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string to = 2;' field")
			}

			cmd.To = FlowID(strings.Clone(v))
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
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
		case 5:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string to = 5;' field")
			}

			cmd.To = FlowID(strings.Clone(v))
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		}
	}

	return nil
}

func unmarshalStackCommand(src []byte, cmd *StackCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message StackCommand {
		//  StateCtxRef carrier_state_ref = 1;
		//  StateCtxRef stack_state_ref = 2;
		//  string annotation = 3;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateCtxRef carrier_state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef carrier_state_ref = 1;' field: %w", err)
			}
			cmd.CarrierStateCtx = stateCtx
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateCtxRef stack_state_ref = 2;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef stack_state_ref = 2;' field: %w", err)
			}
			cmd.StackedStateCtx = stateCtx
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

func unmarshalUnstackCommand(src []byte, cmd *UnstackCommand, stateCtxs stateCtxs) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message UnstackCommand {
		//  StateCtxRef carrier_state_ref = 1;
		//  StateCtxRef unstack_state_ref = 2;
		//  string annotation = 3;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateCtxRef carrier_state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef carrier_state_ref = 1;' field: %w", err)
			}
			cmd.CarrierStateCtx = stateCtx
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateCtxRef unstack_state_ref = 2;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef unstack_state_ref = 2;' field: %w", err)
			}
			cmd.UnstackStateCtx = stateCtx
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

func unmarshalAttachDataCommand(src []byte, cmd *AttachDataCommand, stateCtxs stateCtxs, datas datas) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message AttachDataCommand {
		//  StateCtxRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string alias = 3;
		//  bool store = 4;
		// }

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field")
			}

			d, err := unmarshalDataRef(data, datas)
			if err != nil {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field: %w", err)
			}
			cmd.Data = d
		case 3:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string annotation = 3;' field")
			}

			cmd.Alias = strings.Clone(v)
		case 4:
			v, ok := fc.Bool()
			if !ok {
				return fmt.Errorf("cannot read 'bool store = 4;' field")
			}
			cmd.Store = v
		}
	}

	return nil
}

func unmarshalGetDataCommand(src []byte, cmd *GetDataCommand, stateCtxs stateCtxs, datas datas) (err error) {
	fc := easyproto.FieldContext{}
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field")
		}

		// message GetDataCommand {
		//  StateCtxRef state_ref = 1;
		//  DataRef data_ref = 2;
		//  string alias = 3;
		//}

		switch fc.FieldNum {
		case 1:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		case 2:
			data, ok := fc.MessageData()
			if !ok {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field")
			}

			d, err := unmarshalDataRef(data, datas)
			if err != nil {
				return fmt.Errorf("cannot read 'DataRef data_ref = 1;' field: %w", err)
			}
			cmd.Data = d
		case 3:
			v, ok := fc.String()
			if !ok {
				return fmt.Errorf("cannot read 'string annotation = 3;' field")
			}

			cmd.Alias = strings.Clone(v)
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
		//  StateCtxRef state_ref = 3;
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 3;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 3;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
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
		//  StateCtxRef state_ref = 2;
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 2;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 2;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
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
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field")
			}

			stateCtx, err := unmarshalStateCtxRef(data, stateCtxs)
			if err != nil {
				return fmt.Errorf("cannot read 'StateCtxRef state_ref = 1;' field: %w", err)
			}
			cmd.StateCtx = stateCtx
		}
	}

	return nil
}

//	message StateCtxRef {
//	 int64 idx = 1;
//	}
func marshalStateCtxRef(stateCtx *StateCtx, stateCtxs stateCtxs, mm *easyproto.MessageMarshaler) {
	mm.AppendInt64(1, stateCtxs.idx(stateCtx))
}

func unmarshalStateCtxRef(src []byte, stateCtxs stateCtxs) (*StateCtx, error) {
	var fc easyproto.FieldContext

	var err error
	idx := int64(-1)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return nil, fmt.Errorf("cannot read next field: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			v, ok := fc.Int64()
			if !ok {
				return nil, fmt.Errorf("cannot read 'int idx = 1;' field")
			}
			idx = v
			break
		}
	}

	return stateCtxs.find(idx), nil
}

//	message DataRef {
//	 int64 idx = 1;
//	}
func marshalDataRef(data *Data, datas datas, mm *easyproto.MessageMarshaler) {
	mm.AppendInt64(1, datas.idx(data))
}

func unmarshalDataRef(src []byte, datas datas) (*Data, error) {
	var fc easyproto.FieldContext

	var err error
	idx := int64(-1)
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return nil, fmt.Errorf("cannot read next field: %w", err)
		}
		switch fc.FieldNum {
		case 1:
			v, ok := fc.Int64()
			if !ok {
				return nil, fmt.Errorf("cannot read 'int64 idx = 1;' field")
			}
			idx = v
			break
		}
	}

	return datas.find(idx), nil
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
	case *StackCommand:
		if cmd.CarrierStateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.CarrierStateCtx)
		}
		if cmd.StackedStateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StackedStateCtx)
		}
	case *UnstackCommand:
		if cmd.CarrierStateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.CarrierStateCtx)
		}
		if cmd.UnstackStateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.UnstackStateCtx)
		}
	case *CommitCommand:
		for _, subCmd := range cmd.Commands {
			stateCtxs = append(stateCtxs, commandStateCtxs(subCmd)...)
		}
	case *AttachDataCommand:
		if cmd.StateCtx != nil {
			stateCtxs = append(stateCtxs, cmd.StateCtx)
		}
	case *GetDataCommand:
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

	stateCtxs = slices.CompactFunc(stateCtxs, func(l, r *StateCtx) bool {
		return l == r
	})

	return stateCtxs
}

func commandDatas(cmd0 Command) []*Data {
	datas := make([]*Data, 0)

	switch cmd := cmd0.(type) {
	case *AttachDataCommand:
		if cmd.Data != nil {
			datas = append(datas, cmd.Data)
		}
	case *GetDataCommand:
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

	datas = slices.CompactFunc(datas, func(l, r *Data) bool {
		return l == r
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

func (s stateCtxs) find(idx int64) *StateCtx {
	return s[idx]
}

func (s stateCtxs) findStrIdx(strIdx string) *StateCtx {
	if strIdx == "" {
		return s.find(0)
	}

	idx, err := strconv.ParseInt(strIdx, 10, 64)
	if err != nil {
		idx = -1
	}
	return s.find(idx)
}

func (s stateCtxs) idx(stateCtx *StateCtx) int64 {
	for i, ctx := range s {
		if ctx == stateCtx {
			return int64(i)
		}
	}

	panic(fmt.Errorf("cannot find idx for stateCtx %+v", stateCtx))
}

func (s stateCtxs) strIdx(stateCtx *StateCtx) string {
	return strconv.FormatInt(s.idx(stateCtx), 10)
}

type datas []*Data

func (d datas) find(idx int64) *Data {
	return d[idx]
}

func (d datas) findStrIdx(strIdx string) *Data {
	if strIdx == "" {
		return d.find(0)
	}

	idx, err := strconv.ParseInt(strIdx, 10, 64)
	if err != nil {
		idx = -1
	}
	return d.find(idx)
}

func (d datas) idx(data *Data) int64 {
	for i, data1 := range d {
		if data1 == data {
			return int64(i)
		}
	}

	panic(fmt.Errorf("cannot find idx for data %+v", data))
}

func (d datas) strIdx(data *Data) string {
	return strconv.FormatInt(d.idx(data), 10)
}
