package netflow_test

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/netflow"
	"github.com/thejerf/slogassert"
)

func TestFlowExecute(t *testing.T) {
	lh := slogassert.New(t, slog.LevelDebug, nil)
	l := slog.New(slogassert.New(t, slog.LevelDebug, lh))

	d := memdriver.New(l)
	fr := &flowstate.DefaultFlowRegistry{}
	executedCh := make(chan struct{})
	if err := fr.SetFlow(`aFlow`, flowstate.FlowFunc(func(stateCtx *flowstate.StateCtx, _ flowstate.Engine) (flowstate.Command, error) {
		close(executedCh)
		return flowstate.Park(stateCtx), nil
	})); err != nil {
		t.Fatalf("failed to set flow: %v", err)
	}

	e, err := flowstate.NewEngine(d, fr, l)
	if err != nil {
		t.Fatalf("failed to create flowstate engine: %v", err)
	}

	srv := startSrv(t, e)

	f := netflow.New(srv.URL)

	stateCtx := &flowstate.StateCtx{
		Current: flowstate.State{
			ID: `anID`,
			Transition: flowstate.Transition{
				To: `aFlow`,
			},
		},
	}

	cmd0, err := f.Execute(stateCtx, e)
	if err != nil {
		t.Fatalf("failed to execute flow: %v", err)
	}
	if cmd0 == nil {
		t.Fatal("expected command, got nil")
	}

	cmd, ok := cmd0.(*flowstate.NoopCommand)
	if !ok {
		t.Fatalf("expected NoopCommand, got %T", cmd0)
	}
	if cmd.StateCtx.Current.ID != `anID` {
		t.Fatalf("expected state ID to be 'anID', got '%s'", cmd.StateCtx.Current.ID)
	}

	select {
	case <-executedCh:
		// OK
	case <-time.After(time.Second * 3):
		t.Fatal("remote flow was not executed")
	}
}

func startSrv(t *testing.T, e flowstate.Engine) *httptest.Server {

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == `/health` {
			rw.WriteHeader(http.StatusOK)
			return
		}
		if netflow.HandleExecute(rw, r, e) {
			return
		}

		http.Error(rw, fmt.Sprintf("path %s not supported", r.URL.Path), http.StatusNotFound)
	}))

	t.Cleanup(srv.Close)

	timeoutT := time.NewTimer(time.Second)
	defer timeoutT.Stop()
	readyT := time.NewTicker(time.Millisecond * 50)
	defer readyT.Stop()

loop:
	for {
		select {
		case <-timeoutT.C:
			t.Fatalf("app not ready within %s", time.Second)
		case <-readyT.C:

			resp, err := http.Get(srv.URL + `/health`)
			if err != nil {
				continue loop
			}
			resp.Body.Close()

			return srv
		}
	}
}
