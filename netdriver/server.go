package netdriver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/VictoriaMetrics/easyproto"
	"github.com/makasim/flowstate"
)

func HandleAll(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if HandleGetStateByID(rw, r, d, l) {
		return true
	}
	if HandleGetStateByLabels(rw, r, d, l) {
		return true
	}
	if HandleCommit(rw, r, d, l) {
		return true
	}
	if HandleGetStates(rw, r, d, l) {
		return true
	}
	if HandleGetDelayedStates(rw, r, d, l) {
		return true
	}
	if HandleDelay(rw, r, d, l) {
		return true
	}
	if HandleGetData(rw, r, d, l) {
		return true
	}
	if HandleStoreData(rw, r, d, l) {
		return true
	}

	return false
}

func HandleGetStateByID(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetStateByID" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.GetStateByIDCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	if err := cmd.Prepare(); err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.GetStateByID(cmd); errors.Is(err, flowstate.ErrNotFound) {
		writeNotFoundError(rw, err.Error(), proto)
		return true
	} else if err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func HandleGetStateByLabels(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetStateByLabels" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.GetStateByLabelsCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.GetStateByLabels(cmd); errors.Is(err, flowstate.ErrNotFound) {
		writeNotFoundError(rw, err.Error(), proto)
		return true
	} else if err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func HandleGetStates(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetStates" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.GetStatesCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	cmd.Prepare()

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.GetStates(cmd); err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func HandleGetDelayedStates(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetDelayedStates" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.GetDelayedStatesCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	cmd.Prepare()

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.GetDelayedStates(cmd); err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func HandleDelay(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/Delay" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.DelayCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	if err := cmd.Prepare(); err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.Delay(cmd); err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func HandleCommit(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/Commit" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.CommitCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.Commit(cmd); flowstate.IsErrRevMismatch(err) {
		writeAbortedError(rw, err.Error(), proto)
		return true
	} else if errors.Is(err, flowstate.ErrNotFound) {
		writeNotFoundError(rw, err.Error(), proto)
		return true
	} else if err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func HandleGetData(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetData" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.GetDataCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	if err := cmd.Prepare(); err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.GetData(cmd); err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func HandleStoreData(rw http.ResponseWriter, r *http.Request, d flowstate.Driver, l *slog.Logger) bool {
	if r.URL.Path != "/flowstate.v1.Driver/StoreData" {
		return false
	}

	proto := r.Header.Get("Content-Type") != "application/json"

	cmd, err := readCmd[*flowstate.AttachDataCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	if err := cmd.Prepare(); err != nil {
		writeInvalidArgumentError(rw, err.Error(), proto)
		return true
	}

	flowstate.LogCommand("netdriver", cmd, l)
	if err := d.StoreData(cmd); err != nil {
		writeUnknownError(rw, err.Error(), proto)
		return true
	}

	writeCmd(rw, cmd, proto)
	return true
}

func readCmd[T flowstate.Command](r *http.Request) (T, error) {
	var defCmd T

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return defCmd, fmt.Errorf("failed to read request body: %w", err)
	}

	var cmd0 flowstate.Command
	if r.Header.Get("Content-Type") == "application/json" {
		var err error
		cmd0, err = flowstate.UnmarshalJSONCommand(reqBody)
		if err != nil {
			return defCmd, fmt.Errorf("failed to unmarshal JSON command: %w", err)
		}
	} else {
		var err error
		cmd0, err = flowstate.UnmarshalCommand(reqBody)
		if err != nil {
			return defCmd, fmt.Errorf("failed to unmarshal command: %w", err)
		}
	}

	cmd, ok := cmd0.(T)
	if !ok {
		return defCmd, fmt.Errorf("invalid command type: expected %T; got: %T", defCmd, cmd0)
	}

	return cmd, nil
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

func writeNotFoundError(rw http.ResponseWriter, message string, proto bool) {
	if proto {
		rw.Header().Set("Content-Type", "application/proto")
	} else {
		rw.Header().Set("Content-Type", "application/json")
	}

	rw.WriteHeader(http.StatusNotFound)

	if proto {
		_, _ = rw.Write(marshalError("not_found", message))
	} else {
		_, _ = rw.Write(marshalJSONError("not_found", message))
	}
}

func writeAbortedError(rw http.ResponseWriter, message string, proto bool) {
	if proto {
		rw.Header().Set("Content-Type", "application/proto")
	} else {
		rw.Header().Set("Content-Type", "application/json")
	}

	rw.WriteHeader(http.StatusConflict)

	if proto {
		_, _ = rw.Write(marshalError("aborted", message))
	} else {
		_, _ = rw.Write(marshalJSONError("aborted", message))
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

func unmarshalError(src []byte) (code, message string, err error) {
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return "", "", fmt.Errorf("cannot read next field")
		}
		switch fc.FieldNum {
		case 1:
			v, ok := fc.String()
			if !ok {
				return "", "", fmt.Errorf("cannot read code field")
			}
			code = strings.Clone(v)
		case 2:
			v, ok := fc.String()
			if !ok {
				return "", "", fmt.Errorf("cannot read message field")
			}
			message = strings.Clone(v)
		}
	}

	return code, message, nil
}
