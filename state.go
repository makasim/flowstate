package flowstate

import (
	"context"
	"encoding/json"
	"time"
)

var _ context.Context = &StateCtx{}

type StateID string

type State struct {
	ID          StateID
	Rev         int64
	Annotations map[string]string
	Labels      map[string]string

	CommittedAt time.Time

	Transition Transition
}

func (s State) Annotation(name string) string {
	if value, ok := s.Transition.Annotations[name]; ok {
		return value
	}

	return s.Annotations[name]
}

func (s *State) CopyTo(to *State) State {
	to.ID = s.ID
	to.Rev = s.Rev
	to.CommittedAt = s.CommittedAt
	s.Transition.CopyTo(&to.Transition)

	for k, v := range s.Annotations {
		to.SetAnnotation(k, v)
	}
	for k, v := range s.Labels {
		to.SetLabel(k, v)
	}

	return *to
}

func (s *State) CopyToCtx(to *StateCtx) *StateCtx {
	if s.Rev > 0 {
		s.CopyTo(&to.Committed)
	}
	s.CopyTo(&to.Current)
	return to
}

func (s *State) SetAnnotation(name, value string) {
	if s.Annotations == nil {
		s.Annotations = make(map[string]string)
	}
	s.Annotations[name] = value
}

func (s *State) SetLabel(name, value string) {
	if s.Labels == nil {
		s.Labels = make(map[string]string)
	}
	s.Labels[name] = value
}

func (s State) MarshalJSON() ([]byte, error) {
	jsonS := &jsonState{}
	jsonS.fromState(s)
	return json.Marshal(jsonS)
}

func (s *State) UnmarshalJSON(data []byte) error {
	jsonS := &jsonState{}
	if err := json.Unmarshal(data, jsonS); err != nil {
		return err
	}

	return jsonS.toState(s)
}

type StateCtx struct {
	noCopy noCopy

	Current   State
	Committed State

	// Transitions between committed and current states
	Transitions []Transition

	sessID int64
	e      Engine
	doneCh chan struct{}
}

func (s *StateCtx) SessID() int64 {
	return s.sessID
}

func (s *StateCtx) CopyTo(to *StateCtx) *StateCtx {
	s.Current.CopyTo(&to.Current)
	s.Committed.CopyTo(&to.Committed)

	if cap(to.Transitions) >= len(s.Transitions) {
		to.Transitions = to.Transitions[:len(s.Transitions)]
	} else {
		to.Transitions = make([]Transition, len(s.Transitions))
	}
	for idx := range s.Transitions {
		s.Transitions[idx].CopyTo(&to.Transitions[idx])
	}

	return to
}

func (s *StateCtx) NewTo(id StateID, to *StateCtx) *StateCtx {
	s.CopyTo(to)
	to.Current.ID = id
	to.Current.Rev = 0
	to.Current.ID = id
	to.Current.Rev = 0
	return to
}

func (s *StateCtx) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (s *StateCtx) Done() <-chan struct{} {
	if s.e == nil {
		return nil
	}

	return s.doneCh
}

func (s *StateCtx) Err() error {
	if s.e == nil {
		return nil
	}

	select {
	case <-s.doneCh:
		return context.Canceled
	default:
		return nil
	}
}

func (s *StateCtx) Value(key any) any {
	key1, ok := key.(string)
	if !ok {
		return nil
	}

	return s.Current.Annotations[key1]
}

func (s *StateCtx) MarshalJSON() ([]byte, error) {
	jsonStateCtx := &jsonStateCtx{}
	jsonStateCtx.fromStateCtx(s)
	return json.Marshal(jsonStateCtx)
}

func (s *StateCtx) UnmarshalJSON(data []byte) error {
	jsonStateCtx := &jsonStateCtx{}
	if err := json.Unmarshal(data, jsonStateCtx); err != nil {
		return err
	}
	return jsonStateCtx.toStateCtx(s)
}

type Transition struct {
	To          FlowID
	Annotations map[string]string
}

func (ts *Transition) SetAnnotation(name, value string) {
	if ts.Annotations == nil {
		ts.Annotations = make(map[string]string)
	}
	ts.Annotations[name] = value
}

func (ts *Transition) CopyTo(to *Transition) {
	to.To = ts.To

	if len(ts.Annotations) > 0 {
		if to.Annotations == nil {
			to.Annotations = make(map[string]string)
		}

		for k, v := range ts.Annotations {
			to.Annotations[k] = v
		}
	}
}

func (ts *Transition) String() string {
	return string(ts.To)
}

func (ts *Transition) MarshalJSON() ([]byte, error) {
	jsonTs := &jsonTransition{}
	jsonTs.fromTransition(ts)
	return json.Marshal(jsonTs)
}

func (ts *Transition) UnmarshalJSON(data []byte) error {
	jsonTs := &jsonTransition{}
	if err := json.Unmarshal(data, jsonTs); err != nil {
		return err
	}

	jsonTs.toTransition(ts)
	return nil
}
