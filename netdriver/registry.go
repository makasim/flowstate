package netdriver

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/netflow"
)

type flowRegistry struct {
	mux      sync.Mutex
	httpHost string
	e        flowstate.Engine
	flows    map[flowstate.TransitionID]flowstate.Flow

	hostFlows map[string]flowstate.Flow
}

func NewFlowRegistry(httpHost string) *flowRegistry {
	return &flowRegistry{
		httpHost: httpHost,
		flows:    make(map[flowstate.TransitionID]flowstate.Flow),
	}
}

func (fr *flowRegistry) Init(e flowstate.Engine) error {
	fr.e = e
	return nil
}

func (fr *flowRegistry) Flow(id flowstate.TransitionID) (flowstate.Flow, error) {
	fr.mux.Lock()
	defer fr.mux.Unlock()

	if id == "" {
		return nil, fmt.Errorf("flow id empty")
	}

	if fr.flows == nil {
		return nil, flowstate.ErrFlowNotFound
	}

	f, ok := fr.flows[id]
	if !ok {
		return nil, flowstate.ErrFlowNotFound
	}

	return f, nil
}

func (fr *flowRegistry) SetFlow(id flowstate.TransitionID, flow flowstate.Flow) error {
	fr.mux.Lock()
	defer fr.mux.Unlock()

	if fr.flows == nil {
		fr.flows = make(map[flowstate.TransitionID]flowstate.Flow)
	}

	fr.flows[id] = flow

	stateCtx := &flowstate.StateCtx{}
	err := fr.e.Do(flowstate.GetStateByID(stateCtx, flowstate.StateID(`flowstate.flow.`+string(id)), 0))
	if errors.Is(err, flowstate.ErrNotFound) {
		// ok
	} else if err != nil {
		return fmt.Errorf("get state by id: %w", err)
	}

	if stateCtx.Current.Annotations[`flowstate.flow.http_host`] == fr.httpHost {
		return nil
	}

	stateCtx.Current.Labels[`flow.type`] = "remote"
	stateCtx.Current.SetAnnotation(`flowstate.flow.transition_id`, string(id))
	stateCtx.Current.SetAnnotation(`flowstate.flow.http_host`, fr.httpHost)

	if err := fr.e.Do(flowstate.Commit(flowstate.Pause(stateCtx))); flowstate.IsErrRevMismatch(err) {
		// ok
		return nil
	} else if err != nil {
		return fmt.Errorf("commit flow state: %w", err)
	}

	return nil
}

func (fr *flowRegistry) watchRemoteFlows() {
	t := time.NewTicker(time.Second * 10)

	getStatesCmd := flowstate.GetStatesByLabels(map[string]string{`flow.type`: `remote`})

	for {
		select {
		case <-t.C:
			if err := fr.e.Do(getStatesCmd); err != nil {
				log.Println("watch remote flows: ", err)
			}

			res := getStatesCmd.MustResult()

			for _, state := range res.States {
				getStatesCmd.SinceRev = state.Rev

				if state.Annotations[`flowstate.flow.http_host`] == `` {
					// TODO: log invalid remote state
					continue
				}
				flowHttpHost := state.Annotations[`flowstate.flow.http_host`]

				// local flow, skip
				if flowHttpHost == fr.httpHost {
					continue
				}

				if state.Annotations[`flowstate.flow.transition_id`] == `` {
					// TODO: log invalid remote state
					continue
				}
				flowTsID := flowstate.TransitionID(state.Annotations[`flowstate.flow.transition_id`])

				fr.mux.Lock()

				if flowstate.Ended(state) {
					delete(fr.flows, flowTsID)
					fr.mux.Lock()
					continue
				}

				if fr.hostFlows == nil {
					fr.hostFlows = make(map[string]flowstate.Flow)
				}

				f, ok := fr.hostFlows[flowHttpHost]
				if !ok {
					f = netflow.New(flowHttpHost)
					fr.hostFlows[flowHttpHost] = f
				}

				fr.flows[flowTsID] = f

				fr.mux.Unlock()
			}

			if res.More {
				continue
			}
		}

	}
}
