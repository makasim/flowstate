package netdriver

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/memdriver"
	"github.com/makasim/flowstate/testcases"
)

func TestSuite(t *testing.T) {
	s := testcases.Get(func(t *testing.T) flowstate.Driver {
		l, _ := testcases.NewTestLogger(t)

		srv := startSrv(t, l)

		return New(srv.URL)
	})

	//s.SetUpDelayer = false
	//s.DisableGoleak()
	s.Test(t)
}

func startSrv(t *testing.T, l *slog.Logger) *httptest.Server {
	d := memdriver.New(l)

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path == `/health` {
			rw.WriteHeader(http.StatusOK)
			return
		}
		if HandleAll(rw, r, d) {
			return
		}

		writeNotFoundError(rw, fmt.Sprintf("path %s not found", r.URL.Path))
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
