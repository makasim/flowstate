package netflowstate

import (
	"io"
	"net/http"

	"github.com/makasim/flowstate"
)

func HandleDo(r http.Request, rw http.ResponseWriter, e flowstate.Engine) bool {
	if r.URL.Path != "/flowstate.v1.Engine/Do" {
		return false
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, "failed to read request body: "+err.Error(), http.StatusBadRequest)
		return true
	}

	cmd, err := flowstate.UnmarshalCommand(reqBody)
	if err != nil {
		http.Error(rw, "failed to unmarshal command: "+err.Error(), http.StatusBadRequest)
		return true
	}

	// TODO: handle not found error, rev mismatch error
	if err := e.Do(cmd); err != nil {
		http.Error(rw, "failed to execute command: "+err.Error(), http.StatusInternalServerError)
		return true
	}

	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusOK)

	if _, err := rw.Write(flowstate.MarshalCommand(cmd, nil)); err != nil {
		http.Error(rw, "failed to write response: "+err.Error(), http.StatusInternalServerError)
		return true
	}

	return true
}
