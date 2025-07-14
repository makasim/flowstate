package flowstate_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/makasim/flowstate"
)

func TestMarshalUnmarshalState(t *testing.T) {
	f := func(exp flowstate.State) {
		t.Helper()

		b := flowstate.MarshalState(exp, nil)

		var act flowstate.State
		if err := flowstate.UnmarshalState(b, &act); err != nil {
			t.Fatalf("cannot unmarshal flowstate.State: %v", err)
		}

		if !reflect.DeepEqual(exp, act) {
			t.Fatalf("expected: %+v, got: %+v", exp, act)
		}
	}

	// empty
	f(flowstate.State{})

	// id rev
	f(flowstate.State{
		ID:  "theID",
		Rev: 123,
	})

	// all fields
	f(flowstate.State{
		ID:                   "theID",
		Rev:                  123,
		Annotations:          map[string]string{"fooAnnot": "fooVal", "barAnnot": "barVal"},
		Labels:               map[string]string{"fooLabel": "fooVal", "barLabel": "barVal"},
		CommittedAtUnixMilli: 567,
		Transition: flowstate.Transition{
			From: "fromID",
			To:   "toID",
			Annotations: map[string]string{
				"fooTsAnnot": "fooVal",
			},
		},
	})
}

func TestMarshalUnmarshalStateCtx(t *testing.T) {
	f := func(exp *flowstate.StateCtx) {
		t.Helper()

		b := flowstate.MarshalStateCtx(exp, nil)

		act := &flowstate.StateCtx{}
		if err := flowstate.UnmarshalStateCtx(b, act); err != nil {
			t.Fatalf("cannot unmarshal state ctx: %v", err)
		}

		if !reflect.DeepEqual(exp, act) {
			t.Fatalf("expected: %+v, got: %+v", exp, act)
		}
	}

	// empty
	f(&flowstate.StateCtx{})

	// id rev
	f(&flowstate.StateCtx{
		Committed: flowstate.State{
			ID:  "theID",
			Rev: 123,
		},
		Current: flowstate.State{
			ID:  "theID",
			Rev: 234,
		},
	})

	// all fields
	f(&flowstate.StateCtx{
		Committed: flowstate.State{
			ID:                   "theID",
			Rev:                  123,
			Annotations:          map[string]string{"fooAnnot": "fooVal", "barAnnot": "barVal"},
			Labels:               map[string]string{"fooLabel": "fooVal", "barLabel": "barVal"},
			CommittedAtUnixMilli: 567,
			Transition: flowstate.Transition{
				From: "fromID",
				To:   "toID",
				Annotations: map[string]string{
					"fooTsAnnot": "fooVal",
				},
			},
		},
		Current: flowstate.State{
			ID:                   "theID",
			Rev:                  234,
			Annotations:          map[string]string{"fooAnnotCurr": "fooVal", "barAnnotCurr": "barVal"},
			Labels:               map[string]string{"fooLabelCurr": "fooVal", "barLabelCurr": "barVal"},
			CommittedAtUnixMilli: 567,
			Transition: flowstate.Transition{
				From: "fromIDCurr",
				To:   "toIDCurr",
				Annotations: map[string]string{
					"fooTsAnnotCurr": "fooVal",
				},
			},
		},
		Transitions: []flowstate.Transition{
			{
				From: "fromID1",
				To:   "toID1",
			},
			{
				From: "fromID2",
				To:   "toID2",
			},
		},
	})
}

func TestMarshalUnmarshalDelayedState(t *testing.T) {
	f := func(exp flowstate.DelayedState) {
		t.Helper()

		b := flowstate.MarshalDelayedState(exp, nil)

		act := flowstate.DelayedState{}
		if err := flowstate.UnmarshalDelayedState(b, &act); err != nil {
			t.Fatalf("cannot unmarshal delayed state: %v", err)
		}

		if !reflect.DeepEqual(exp, act) {
			t.Fatalf("expected: %+v, got: %+v", exp, act)
		}
	}

	// empty
	f(flowstate.DelayedState{})

	// id rev
	f(flowstate.DelayedState{
		State: flowstate.State{
			ID:  "theID",
			Rev: 123,
		},
	})

	// zero execute at
	f(flowstate.DelayedState{
		State: flowstate.State{
			ID:  "theID",
			Rev: 123,
		},
		Offset: 234,
	})

	// not zero execute at
	f(flowstate.DelayedState{
		State: flowstate.State{
			ID:  "theID",
			Rev: 123,
		},
		Offset:    234,
		ExecuteAt: time.Unix(345, 0),
	})
}

func TestMarshalUnmarshalTransition(t *testing.T) {
	f := func(exp flowstate.Transition) {
		t.Helper()

		b := flowstate.MarshalTransition(exp, nil)

		var act flowstate.Transition
		if err := flowstate.UnmarshalTransition(b, &act); err != nil {
			t.Fatalf("cannot unmarshal state: %v", err)
		}

		if !reflect.DeepEqual(exp, act) {
			t.Fatalf("expected: %+v, got: %+v", exp, act)
		}
	}

	// empty
	f(flowstate.Transition{})

	// id rev
	f(flowstate.Transition{
		From: "fromID",
		To:   "toID",
	})

	// all fields
	f(flowstate.Transition{
		From:        "fromID",
		To:          "toID",
		Annotations: map[string]string{"fooAnnot": "fooVal", "barAnnot": "barVal"},
	})
}

func TestMarshalUnmarshalData(t *testing.T) {
	f := func(exp *flowstate.Data) {
		t.Helper()

		b := flowstate.MarshalData(exp, nil)

		act := &flowstate.Data{}
		if err := flowstate.UnmarshalData(b, act); err != nil {
			t.Fatalf("cannot unmarshal flowstate.Data: %v", err)
		}

		if !reflect.DeepEqual(exp, act) {
			t.Fatalf("expected: %+v, got: %+v", exp, act)
		}
	}

	// empty
	f(&flowstate.Data{})

	// id rev
	f(&flowstate.Data{
		ID:  "theID",
		Rev: 123,
	})

	// string B
	f(&flowstate.Data{
		ID:  "theID",
		Rev: 123,
		B:   []byte("fooValString"),
	})

	// binary B
	f(&flowstate.Data{
		ID:     "theID",
		Rev:    123,
		Binary: true,
		B:      []byte{42, 0, 0, 0, 31, 133, 235, 81, 184, 30, 9, 64},
	})
}

func TestMarshalUnmarshalCommand(t *testing.T) {
	f := func(exp flowstate.Command) {
		t.Helper()

		b := flowstate.MarshalCommand(exp, nil)

		act, err := flowstate.UnmarshalCommand(b)
		if err != nil {
			t.Fatalf("cannot unmarshal flowstate.Command: %v", err)
		}

		if !reflect.DeepEqual(exp, act) {
			t.Fatalf("expected: %+v, got: %+v", exp, act)
		}
	}

	f(&flowstate.TransitCommand{})

	f(&flowstate.TransitCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		To: "theFlowID",
	})

	f(&flowstate.PauseCommand{})

	f(&flowstate.PauseCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		To: "theFlowID",
	})

	f(&flowstate.ResumeCommand{})

	f(&flowstate.ResumeCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
	})

	f(&flowstate.EndCommand{})

	f(&flowstate.EndCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
	})

	f(&flowstate.ExecuteCommand{})

	f(&flowstate.ExecuteCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
	})

	f(&flowstate.DelayCommand{})

	f(&flowstate.DelayCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		DelayingState: flowstate.State{
			ID:  "theDelayingID",
			Rev: 234,
		},
		ExecuteAt: time.Unix(345, 0),
		Commit:    true,
	})

	f(&flowstate.CommitCommand{})

	f(&flowstate.CommitCommand{
		Commands: []flowstate.Command{
			&flowstate.TransitCommand{
				StateCtx: &flowstate.StateCtx{
					Current: flowstate.State{
						ID:  "theTransitID",
						Rev: 123,
					},
				},
				To: "theFlowID",
			},
			&flowstate.PauseCommand{
				StateCtx: &flowstate.StateCtx{
					Current: flowstate.State{
						ID:  "thePauseID",
						Rev: 234,
					},
				},
			},
		},
	})

	f(&flowstate.NoopCommand{})

	f(&flowstate.NoopCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
	})

	f(&flowstate.SerializeCommand{})

	f(&flowstate.SerializeCommand{
		SerializableStateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theSerializableID",
				Rev: 234,
			},
		},
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Annotation: "theAnnotation",
	})

	f(&flowstate.DeserializeCommand{})

	f(&flowstate.DeserializeCommand{
		DeserializedStateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theDeserializedID",
				Rev: 234,
			},
		},
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Annotation: "theAnnotation",
	})

	f(&flowstate.StoreDataCommand{})

	f(&flowstate.StoreDataCommand{
		Data: &flowstate.Data{
			ID:  "theDataID",
			Rev: 123,
			B:   []byte("theDataValue"),
		},
	})

	f(&flowstate.GetDataCommand{})

	f(&flowstate.GetDataCommand{
		Data: &flowstate.Data{
			ID:  "theDataID",
			Rev: 123,
			B:   []byte("theDataValue"),
		},
	})

	f(&flowstate.ReferenceDataCommand{})

	f(&flowstate.ReferenceDataCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Data: &flowstate.Data{
			ID:  "theDataID",
			Rev: 123,
			B:   []byte("theDataValue"),
		},
		Annotation: "theAnnotation",
	})

	f(&flowstate.DereferenceDataCommand{})

	f(&flowstate.DereferenceDataCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Data: &flowstate.Data{
			ID:  "theDataID",
			Rev: 123,
			B:   []byte("theDataValue"),
		},
		Annotation: "theAnnotation",
	})

	f(&flowstate.GetStateByIDCommand{})

	f(&flowstate.GetStateByIDCommand{
		ID:  "theID",
		Rev: 123,
	})

	f(&flowstate.GetStateByIDCommand{
		ID:  "theID",
		Rev: 123,
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
	})

	f(&flowstate.GetStateByLabelsCommand{})

	f(&flowstate.GetStateByLabelsCommand{
		Labels: map[string]string{
			"fooLabel": "fooVal",
			"barLabel": "barVal",
		},
	})

	f(&flowstate.GetStateByLabelsCommand{
		Labels: map[string]string{
			"fooLabel": "fooVal",
			"barLabel": "barVal",
		},
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
	})

	f(&flowstate.GetStatesCommand{})

	f(&flowstate.GetStatesCommand{
		SinceRev:   123,
		SinceTime:  time.Unix(56789, 0),
		LatestOnly: true,
		Limit:      555,
		Labels: []map[string]string{
			{
				"fooLabel": "fooVal",
			},
			{
				"barLabel": "barVal",
			},
		},
	})

	f(&flowstate.GetStatesCommand{
		SinceRev:   123,
		SinceTime:  time.Unix(56789, 0),
		LatestOnly: true,
		Limit:      555,
		Labels: []map[string]string{
			{
				"fooLabel": "fooVal",
			},
			{
				"barLabel": "barVal",
			},
		},

		Result: &flowstate.GetStatesResult{
			States: []flowstate.State{
				{
					ID:  "theID1",
					Rev: 123,
				},
				{
					ID:  "theID2",
					Rev: 234,
				},
			},
			More: true,
		},
	})

	f(&flowstate.GetDelayedStatesCommand{})

	f(&flowstate.GetDelayedStatesCommand{
		Since:  time.Unix(123, 0),
		Until:  time.Unix(234, 0),
		Offset: 345,
		Limit:  567,
	})

	f(&flowstate.GetDelayedStatesCommand{
		Since:  time.Unix(123, 0),
		Until:  time.Unix(234, 0),
		Offset: 345,
		Limit:  567,

		Result: &flowstate.GetDelayedStatesResult{
			States: []flowstate.DelayedState{
				{
					State: flowstate.State{
						ID:  "theID1",
						Rev: 123,
					},
					Offset:    456,
					ExecuteAt: time.Unix(1234, 0),
				},
				{
					State: flowstate.State{
						ID:  "theID2",
						Rev: 234,
					},
					Offset:    567,
					ExecuteAt: time.Unix(12345, 0),
				},
			},
			More: true,
		},
	})

	f(&flowstate.CommitStateCtxCommand{})

	f(&flowstate.CommitStateCtxCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
	})
}
