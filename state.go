package flowstate

import (
	"context"
	"time"
)

var _ context.Context = &StateCtx{}

type StateID string

type State struct {
	ID          StateID           `json:"id,omitempty"`
	Rev         int64             `json:"rev,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`

	CommittedAtUnixMilli int64 `json:"committed_at_unix_milli,omitempty"`

	Transition Transition `json:"transition,omitempty"`
}

func (s *State) SetCommitedAt(at time.Time) {
	s.CommittedAtUnixMilli = at.UnixMilli()
}

func (s *State) CommittedAt() time.Time {
	return time.UnixMilli(s.CommittedAtUnixMilli)
}

func (s *State) CopyTo(to *State) State {
	to.ID = s.ID
	to.Rev = s.Rev
	to.CommittedAtUnixMilli = s.CommittedAtUnixMilli
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

type StateCtx struct {
	noCopy noCopy

	Current   State `json:"current,omitempty"`
	Committed State `json:"committed,omitempty"`

	// Transitions between committed and current states
	Transitions []Transition `json:"transitions,omitempty"`

	sessID int64  `json:"-"`
	e      Engine `json:"-"`
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

type TransitionID string

type Transition struct {
	From        TransitionID      `json:"from,omitempty"`
	To          TransitionID      `json:"to,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

func (ts *Transition) SetAnnotation(name, value string) {
	if ts.Annotations == nil {
		ts.Annotations = make(map[string]string)
	}
	ts.Annotations[name] = value
}

func (ts *Transition) CopyTo(to *Transition) {
	to.From = ts.From
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
	return string(ts.From) + `->` + string(ts.To)
}
