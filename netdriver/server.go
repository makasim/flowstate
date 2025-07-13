package netdriver

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/VictoriaMetrics/easyproto"
	"github.com/makasim/flowstate"
)

func HandleAll(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if HandleGetStateByID(rw, r, d) {
		return true
	}
	if HandleGetStateByLabels(rw, r, d) {
		return true
	}
	if HandleCommit(rw, r, d) {
		return true
	}
	if HandleGetStates(rw, r, d) {
		return true
	}
	if HandleGetDelayedStates(rw, r, d) {
		return true
	}
	if HandleDelay(rw, r, d) {
		return true
	}
	if HandleGetData(rw, r, d) {
		return true
	}
	if HandleStoreData(rw, r, d) {
		return true
	}

	return false
}

func HandleGetStateByID(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetStateByID" {
		return false
	}

	cmd, err := readCmd[*flowstate.GetStateByIDCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.GetStateByID(cmd); errors.Is(err, flowstate.ErrNotFound) {
		writeNotFoundError(rw, err.Error())
		return true
	} else if err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func HandleGetStateByLabels(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetStateByLabels" {
		return false
	}

	cmd, err := readCmd[*flowstate.GetStateByLabelsCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.GetStateByLabels(cmd); errors.Is(err, flowstate.ErrNotFound) {
		writeNotFoundError(rw, err.Error())
		return true
	} else if err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func HandleGetStates(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetStates" {
		return false
	}

	cmd, err := readCmd[*flowstate.GetStatesCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.GetStates(cmd); err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func HandleGetDelayedStates(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetDelayedStates" {
		return false
	}

	cmd, err := readCmd[*flowstate.GetDelayedStatesCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.GetDelayedStates(cmd); err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func HandleDelay(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/Delay" {
		return false
	}

	cmd, err := readCmd[*flowstate.DelayCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.Delay(cmd); err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func HandleCommit(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/Commit" {
		return false
	}

	cmd, err := readCmd[*flowstate.CommitCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.Commit(cmd); flowstate.IsErrRevMismatch(err) {
		writeAbortedError(rw, err.Error())
		return true
	} else if errors.Is(err, flowstate.ErrNotFound) {
		writeNotFoundError(rw, err.Error())
		return true
	} else if err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func HandleGetData(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/GetData" {
		return false
	}

	cmd, err := readCmd[*flowstate.GetDataCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.GetData(cmd); err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func HandleStoreData(rw http.ResponseWriter, r *http.Request, d flowstate.Driver) bool {
	if r.URL.Path != "/flowstate.v1.Driver/StoreData" {
		return false
	}

	cmd, err := readCmd[*flowstate.StoreDataCommand](r)
	if err != nil {
		writeInvalidArgumentError(rw, err.Error())
		return true
	}

	if err := d.StoreData(cmd); err != nil {
		writeUnknownError(rw, err.Error())
		return true
	}

	writeCmd(rw, cmd)
	return true
}

func readCmd[T flowstate.Command](r *http.Request) (T, error) {
	var defCmd T

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		return defCmd, fmt.Errorf("failed to read request body: %w", err)
	}

	cmd0, err := flowstate.UnmarshalCommand(reqBody)
	if err != nil {
		return defCmd, fmt.Errorf("failed to unmarshal command: %w", err)
	}

	cmd, ok := cmd0.(T)
	if !ok {
		return defCmd, fmt.Errorf("invalid command type: expected %T; got: %T", defCmd, cmd0)
	}

	return cmd, nil
}

func writeCmd(rw http.ResponseWriter, cmd flowstate.Command) {
	rw.Header().Set("Content-Type", "application/proto")
	rw.WriteHeader(http.StatusOK)

	_, _ = rw.Write(flowstate.MarshalCommand(cmd, nil))
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
