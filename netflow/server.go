package netflow

import (
	"fmt"
	"io"
	"net/http"

	"github.com/VictoriaMetrics/easyproto"
	"github.com/makasim/flowstate"
)

func HandleExecute(rw http.ResponseWriter, r *http.Request, e flowstate.Engine) bool {
	if r.URL.Path != "/flowstate.v1.Flow/Execute" {
		return false
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeInvalidArgumentError(rw, "failed to read request body: "+err.Error())
		return true
	}

	stateCtx := &flowstate.StateCtx{}
	if err := flowstate.UnmarshalStateCtx(reqBody, stateCtx); err != nil {
		writeInvalidArgumentError(rw, "failed to unmarshal command: "+err.Error())
		return true
	}
	resStateCtx := stateCtx.CopyTo(&flowstate.StateCtx{})

	if err := e.Do(flowstate.Execute(stateCtx)); err != nil {
		writeUnknownError(rw, fmt.Sprintf("failed to execute command: %s", err.Error()))
		return true
	}

	writeCmd(rw, flowstate.Noop(resStateCtx))
	return true
}

func writeCmd(rw http.ResponseWriter, cmd flowstate.Command) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusOK)

	_, _ = rw.Write(flowstate.MarshalCommand(cmd, nil))
}

func writeUnknownError(rw http.ResponseWriter, message string) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusInternalServerError)

	_, _ = rw.Write(marshalError("unknown", message))
}

func writeInvalidArgumentError(rw http.ResponseWriter, message string) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusBadRequest)

	_, _ = rw.Write(marshalError("invalid_argument", message))
}

func marshalError(code, message string) []byte {
	m := &easyproto.Marshaler{}
	mm := m.MessageMarshaler()

	if code != "" {
		mm.AppendString(1, code)
	}
	if message != "" {
		mm.AppendString(2, message)
	}

	return m.Marshal(nil)
}
