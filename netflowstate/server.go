package netflowstate

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/VictoriaMetrics/easyproto"
	"github.com/makasim/flowstate"
)

func HandleGetStateByID(r http.Request, rw http.ResponseWriter, d flowstate.Driver) bool {
	// rpc GetStateByID(Command) returns (Command) {}
	if r.URL.Path != "/flowstate.v1.Driver/GetStateByID" {
		return false
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeInvalidArgumentError(rw, "failed to read request body: "+err.Error())
		return true
	}

	cmd0, err := flowstate.UnmarshalCommand(reqBody)
	if err != nil {
		writeInvalidArgumentError(rw, "failed to unmarshal command: "+err.Error())
		return true
	}
	cmd, ok := cmd0.(*flowstate.GetStateByIDCommand)
	if !ok {
		writeInvalidArgumentError(rw, fmt.Sprintf("invalid command type: expected GetStateByIDCommand; got: %T", cmd0))
		return true
	}

	if err := d.GetStateByID(cmd); flowstate.IsErrRevMismatch(err) {
		writeAbortedError(rw, err.Error())
		return true
	} else if errors.Is(err, flowstate.ErrNotFound) {
		writeNotFoundError(rw, err.Error())
		return true
	} else if err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusOK)

	_, _ = rw.Write(flowstate.MarshalCommand(cmd, nil))
	return true
}

func writeInvalidArgumentError(rw http.ResponseWriter, message string) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusBadRequest)

	_, _ = rw.Write(marshalError("invalid_argument", message))
}

func writeUnknownError(rw http.ResponseWriter, message string) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusInternalServerError)

	_, _ = rw.Write(marshalError("unknown", message))
}

func writeNotFoundError(rw http.ResponseWriter, message string) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusNotFound)

	_, _ = rw.Write(marshalError("not_found", message))
}

func writeAbortedError(rw http.ResponseWriter, message string) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusConflict)

	_, _ = rw.Write(marshalError("aborted", message))
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
