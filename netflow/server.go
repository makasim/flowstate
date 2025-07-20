package netflow

import (
	"encoding/json"
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

	proto := r.Header.Get("Content-Type") != "application/json"

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeInvalidArgumentError(rw, "failed to read request body: "+err.Error(), proto)
		return true
	}

	stateCtx := &flowstate.StateCtx{}
	if proto {
		if err := flowstate.UnmarshalStateCtx(reqBody, stateCtx); err != nil {
			writeInvalidArgumentError(rw, "failed to unmarshal command: "+err.Error(), proto)
			return true
		}
	} else {
		if err = flowstate.UnmarshalJSONStateCtx(reqBody, stateCtx); err != nil {
			writeInvalidArgumentError(rw, "failed to unmarshal JSON command: "+err.Error(), proto)
			return true
		}
	}

	resStateCtx := stateCtx.CopyTo(&flowstate.StateCtx{})

	if err := e.Do(flowstate.Execute(stateCtx)); err != nil {
		writeUnknownError(rw, fmt.Sprintf("failed to execute command: %s", err.Error()), proto)
		return true
	}

	writeCmd(rw, flowstate.Noop(resStateCtx), proto)
	return true
}

func writeCmd(rw http.ResponseWriter, cmd flowstate.Command, proto bool) {
	if proto {
		rw.Header().Set("Content-Type", "application/proto")
	} else {
		rw.Header().Set("Content-Type", "application/json")
	}

	rw.WriteHeader(http.StatusOK)

	if proto {
		_, _ = rw.Write(flowstate.MarshalCommand(cmd, nil))
	} else {
		b, _ := flowstate.MarshalJSONCommand(cmd)
		_, _ = rw.Write(b)
	}
}

func writeUnknownError(rw http.ResponseWriter, message string, proto bool) {
	if proto {
		rw.Header().Set("Content-Type", "application/proto")
	} else {
		rw.Header().Set("Content-Type", "application/json")
	}

	rw.WriteHeader(http.StatusInternalServerError)

	if proto {
		_, _ = rw.Write(marshalError("unknown", message))
	} else {
		_, _ = rw.Write(marshalJSONError("unknown", message))
	}
}

func writeInvalidArgumentError(rw http.ResponseWriter, message string, proto bool) {
	if proto {
		rw.Header().Set("Content-Type", "application/proto")
	} else {
		rw.Header().Set("Content-Type", "application/json")
	}

	rw.WriteHeader(http.StatusBadRequest)

	if proto {
		_, _ = rw.Write(marshalError("invalid_argument", message))
	} else {
		_, _ = rw.Write(marshalJSONError("invalid_argument", message))
	}
}

func marshalJSONError(code, message string) []byte {
	b, _ := json.Marshal(map[string]string{
		"code":    code,
		"message": message,
	})
	return b
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
