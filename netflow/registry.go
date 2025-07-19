package netflow

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.FlowRegistry = (*Registry)(nil)

type Registry struct {
	httpHost string
	fr       *flowstate.DefaultFlowRegistry
	d        flowstate.Driver
	l        *slog.Logger

	hostsFlowsMux sync.Mutex
	hostFlows     map[string]*Flow
	closeCh       chan struct{}
	closedCh      chan struct{}
}

func NewRegistry(httpHost string, d flowstate.Driver, l *slog.Logger) *Registry {
	fr := &Registry{
		d:        d,
		l:        l,
		httpHost: httpHost,

		fr:       &flowstate.DefaultFlowRegistry{},
		closeCh:  make(chan struct{}),
		closedCh: make(chan struct{}),
	}

	go fr.watchFlows()

	return fr
}

func (fr *Registry) Flow(id flowstate.TransitionID) (flowstate.Flow, error) {
	return fr.fr.Flow(id)
}

func (fr *Registry) SetFlow(id flowstate.TransitionID, flow flowstate.Flow) error {
	if err := fr.fr.SetFlow(id, flow); err != nil {
		return err
	}

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: flowStateID(id),
		},
	}
	err := fr.d.GetStateByID(flowstate.GetStateByID(stateCtx, stateCtx.Current.ID, 0))
	if errors.Is(err, flowstate.ErrNotFound) {
		// ok
	} else if err != nil {
		return fmt.Errorf("get state by id: %w", err)
	}

	if stateCtx.Current.Annotations[`flowstate.flow.http_host`] == fr.httpHost {
		return nil
	}

	stateCtx.Current.SetLabel(`flow.type`, `remote`)
	stateCtx.Current.SetAnnotation(`flowstate.flow.transition_id`, string(id))
	stateCtx.Current.SetAnnotation(`flowstate.flow.http_host`, fr.httpHost)

	if err := fr.d.Commit(flowstate.Commit(flowstate.Pause(stateCtx))); flowstate.IsErrRevMismatch(err) {
		// ok
		return nil
	} else if err != nil {
		return fmt.Errorf("commit flow state: %w", err)
	}

	return nil
}

func (fr *Registry) UnsetFlow(id flowstate.TransitionID) error {
	f0, err := fr.fr.Flow(id)
	if errors.Is(err, flowstate.ErrNotFound) {
		return nil
	} else if err != nil {
		return fmt.Errorf("get flow: %w", err)
	}

	if err := fr.fr.UnsetFlow(id); err != nil {
		return err
	}

	if _, ok := f0.(*Flow); ok {
		return nil
	}

	fr.hostsFlowsMux.Lock()
	defer fr.hostsFlowsMux.Unlock()

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: flowStateID(id),
		},
	}
	if err = fr.d.GetStateByID(flowstate.GetStateByID(stateCtx, stateCtx.Current.ID, 0)); err != nil {
		return fmt.Errorf("get state by id: %w", err)
	}

	if err := fr.d.Commit(flowstate.Commit(flowstate.End(stateCtx))); flowstate.IsErrRevMismatch(err) {
		// ok
		return nil
	} else if err != nil {
		return fmt.Errorf("commit flow state: %w", err)
	}

	return nil
}

func (fr *Registry) Close() {
	close(fr.closeCh)
	<-fr.closedCh
}

func (fr *Registry) watchFlows() {
	defer close(fr.closedCh)

	t := time.NewTicker(time.Second * 10)
	defer t.Stop()

	labels := map[string]string{`flow.type`: `remote`}
	getStatesCmd := flowstate.GetStatesByLabels(labels).WithLatestOnly()

	for {
		fr.l.Debug("get states", "labels", labels, "latest_only", true, "since_rev", getStatesCmd.SinceRev)

		if err := fr.d.GetStates(getStatesCmd); err != nil {
			fr.l.Error("driver: get states failed", "error", err)
			continue
		}

		res := getStatesCmd.MustResult()

		for _, state := range res.States {
			getStatesCmd.SinceRev = state.Rev

			if state.Annotations[`flowstate.flow.http_host`] == `` {
				fr.l.Warn("state has no 'flowstate.flow.http_host' annotation set, skipping", "state_id", state.ID, "state_rev", state.Rev)
				continue
			}
			httpHost := state.Annotations[`flowstate.flow.http_host`]

			// local flow, skip
			if httpHost == fr.httpHost {
				continue
			}

			if state.Annotations[`flowstate.flow.transition_id`] == `` {
				fr.l.Warn("flow state has no 'flowstate.flow.transition_id' annotation set, skipping", "state_id", state.ID, "state_rev", state.Rev)
				continue
			}
			tsID := flowstate.TransitionID(state.Annotations[`flowstate.flow.transition_id`])

			if flowstate.Ended(state) {
				if err := fr.fr.UnsetFlow(tsID); err != nil {
					fr.l.Warn("flow registry: unset flow failed", "error", err, "transition_id", tsID, "state_id", state.ID, "state_rev", state.Rev)
				}

				continue
			}

			fr.hostsFlowsMux.Lock()

			if fr.hostFlows == nil {
				fr.hostFlows = make(map[string]*Flow)
			}

			f, ok := fr.hostFlows[httpHost]
			if !ok {
				f = New(httpHost)
				fr.hostFlows[httpHost] = f
			}

			fr.hostsFlowsMux.Unlock()

			if err := fr.fr.SetFlow(tsID, f); err != nil {
				fr.l.Warn("flow registry: set flow failed", "error", err, "transition_id", tsID, "state_id", state.ID, "state_rev", state.Rev)
			}
		}

		if res.More {
			continue
		}

		select {
		case <-t.C:
			continue
		case <-fr.closeCh:
			return
		}
	}
}

func flowStateID(tsID flowstate.TransitionID) flowstate.StateID {
	return flowstate.StateID(`flowstate.flow.` + string(tsID))
}
