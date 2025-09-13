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
		ID:          "theID",
		Rev:         123,
		Annotations: map[string]string{"fooAnnot": "fooVal", "barAnnot": "barVal"},
		Labels:      map[string]string{"fooLabel": "fooVal", "barLabel": "barVal"},
		CommittedAt: time.UnixMilli(567),
		Transition: flowstate.Transition{
			To: "toID",
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
			ID:          "theID",
			Rev:         123,
			Annotations: map[string]string{"fooAnnot": "fooVal", "barAnnot": "barVal"},
			Labels:      map[string]string{"fooLabel": "fooVal", "barLabel": "barVal"},
			CommittedAt: time.UnixMilli(567),
			Transition: flowstate.Transition{
				To: "toID",
				Annotations: map[string]string{
					"fooTsAnnot": "fooVal",
				},
			},
		},
		Current: flowstate.State{
			ID:          "theID",
			Rev:         234,
			Annotations: map[string]string{"fooAnnotCurr": "fooVal", "barAnnotCurr": "barVal"},
			Labels:      map[string]string{"fooLabelCurr": "fooVal", "barLabelCurr": "barVal"},
			CommittedAt: time.UnixMilli(567),
			Transition: flowstate.Transition{
				To: "toIDCurr",
				Annotations: map[string]string{
					"fooTsAnnotCurr": "fooVal",
				},
			},
		},
		Transitions: []flowstate.Transition{
			{
				To: "toID1",
			},
			{
				To: "toID2",
			},
		},
		Datas: map[string]*flowstate.Data{
			"theBinaryData": {
				Rev: 123,
				Annotations: map[string]string{
					"binary": "true",
				},
				Blob: []byte("theBinaryDataValue"),
			},
			"theStringData": {
				Rev:  321,
				Blob: []byte("theStringDataValue"),
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
		To: "toID",
	})

	// all fields
	f(flowstate.Transition{
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

	// id rev annotations
	f(&flowstate.Data{
		Rev: 123,
		Annotations: map[string]string{
			"fooAnnot": "fooVal",
			"barAnnot": "barVal",
		},
	})

	// string Blob
	f(&flowstate.Data{
		Rev:  123,
		Blob: []byte("fooValString"),
	})

	// binary Blob
	f(&flowstate.Data{
		Rev: 123,
		Annotations: map[string]string{
			"binary": "true",
		},
		Blob: []byte{42, 0, 0, 0, 31, 133, 235, 81, 184, 30, 9, 64},
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
		Annotations: map[string]string{
			"fooTsAnnot": "fooVal",
			"barTsAnnot": "barVal",
		},
	})

	f(&flowstate.ParkCommand{})

	f(&flowstate.ParkCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Annotations: map[string]string{
			"fooTsAnnot": "fooVal",
			"barTsAnnot": "barVal",
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
		Result: &flowstate.DelayedState{
			State: flowstate.State{
				ID:  "theDelayingID",
				Rev: 234,
			},
			Offset:    678,
			ExecuteAt: time.Unix(345, 0),
		},
		ExecuteAt: time.Unix(345, 0),
		Commit:    true,
		To:        "theFlowID",
		Annotations: map[string]string{
			"fooTsAnnot": "fooVal",
			"barTsAnnot": "barVal",
		},
	})

	f(&flowstate.CommitCommand{})

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "theTransitID",
			Rev: 123,
		},
	}
	f(&flowstate.CommitCommand{
		Commands: []flowstate.Command{
			&flowstate.TransitCommand{
				StateCtx: stateCtx,
			},
			&flowstate.ParkCommand{
				StateCtx: stateCtx,
			},
		},
	})

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
			&flowstate.ParkCommand{
				StateCtx: &flowstate.StateCtx{
					Current: flowstate.State{
						ID:  "thePauseID",
						Rev: 234,
					},
				},
			},
		},
	})

	f(&flowstate.CommitCommand{
		Commands: []flowstate.Command{
			&flowstate.StoreDataCommand{
				StateCtx: &flowstate.StateCtx{
					Current: flowstate.State{
						ID:  "theStoreDataID",
						Rev: 123,
					},
				},
				Alias: "foo",
			},
			&flowstate.StoreDataCommand{
				StateCtx: &flowstate.StateCtx{
					Current: flowstate.State{
						ID:  "theOtherStoreDataID",
						Rev: 321,
					},
				},
				Alias: "bar",
			},
		},
	})

	f(&flowstate.CommitCommand{
		Commands: []flowstate.Command{
			&flowstate.GetDataCommand{
				StateCtx: &flowstate.StateCtx{
					Current: flowstate.State{
						ID:  "theStoreDataID",
						Rev: 123,
					},
				},
				Alias: "foo",
			},
			&flowstate.GetDataCommand{
				StateCtx: &flowstate.StateCtx{
					Current: flowstate.State{
						ID:  "theOtherStoreDataID",
						Rev: 321,
					},
				},
				Alias: "bar",
			},
		},
	})

	f(&flowstate.NoopCommand{})

	f(&flowstate.StackCommand{})

	f(&flowstate.StackCommand{
		StackedStateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theSerializableID",
				Rev: 234,
			},
		},
		CarrierStateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Annotation: "theAnnotation",
	})

	f(&flowstate.UnstackCommand{})

	f(&flowstate.UnstackCommand{
		UnstackStateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theUnstackID",
				Rev: 234,
			},
		},
		CarrierStateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Annotation: "theAnnotation",
	})

	f(&flowstate.StoreDataCommand{})

	f(&flowstate.StoreDataCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Alias: "foo",
	})

	f(&flowstate.GetDataCommand{})

	f(&flowstate.GetDataCommand{
		StateCtx: &flowstate.StateCtx{
			Current: flowstate.State{
				ID:  "theID",
				Rev: 123,
			},
		},
		Alias: "foo",
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
}

func TestMarshalUnmarshalCommandPreserveRef(t *testing.T) {
	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID:  "theID",
			Rev: 123,
		},
	}

	exp := flowstate.Commit(
		flowstate.Transit(stateCtx, `foo`),
		flowstate.Park(stateCtx),
		flowstate.Stack(stateCtx, stateCtx, `foo`),
		flowstate.Unstack(stateCtx, stateCtx, `foo`),

		// TODO: add more commands to test
	)
	b := flowstate.MarshalCommand(exp, nil)

	act, err := flowstate.UnmarshalCommand(b)
	if err != nil {
		t.Fatalf("cannot unmarshal flowstate.Command: %v", err)
	}

	if !reflect.DeepEqual(exp, act) {
		t.Fatalf("expected: %+v, got: %+v", exp, act)
	}

	actCommit := act.(*flowstate.CommitCommand)
	actTransit := actCommit.Commands[0].(*flowstate.TransitCommand)
	actStateCtx := actTransit.StateCtx
	if actStateCtx == nil {
		t.Fatal("expected non-nil StateCtx in TransitCommand")
	}

	actPark := actCommit.Commands[1].(*flowstate.ParkCommand)
	if actPark.StateCtx != actStateCtx {
		t.Fatalf("expected StateCtx in ParkCommand to be the same ref stateCtx")
	}

	actStack := actCommit.Commands[2].(*flowstate.StackCommand)
	if actStack.CarrierStateCtx != actStateCtx {
		t.Fatalf("expected StateCtx in StackCommand to be the same ref stateCtx")
	}
	if actStack.StackedStateCtx != actStateCtx {
		t.Fatalf("expected StateCtx in StackCommand to be the same ref stateCtx")
	}

	actUnstack := actCommit.Commands[3].(*flowstate.UnstackCommand)
	if actUnstack.CarrierStateCtx != actStateCtx {
		t.Fatalf("expected StateCtx in UnstackCommand to be the same ref stateCtx")
	}
	if actUnstack.UnstackStateCtx != actStateCtx {
		t.Fatalf("expected StateCtx in UnstackCommand to be the same ref stateCtx")
	}
}
