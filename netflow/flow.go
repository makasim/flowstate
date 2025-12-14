package netflow

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/VictoriaMetrics/easyproto"
	"github.com/makasim/flowstate"
)

type Config struct {
}

type Flow struct {
	httpHost string

	c *http.Client
}

func New(httpHost string) *Flow {
	return &Flow{
		httpHost: httpHost,

		c: &http.Client{},
	}
}

func (f *Flow) Execute(stateCtx *flowstate.StateCtx, _ *flowstate.Engine) (flowstate.Command, error) {
	b := flowstate.MarshalStateCtx(stateCtx, nil)
	req, err := http.NewRequest(`POST`, strings.TrimRight(f.httpHost, `/`)+`/flowstate.v1.Flow/Execute`, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := f.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if http.StatusOK != resp.StatusCode {
		code, message, err := unmarshalError(b)
		if err != nil {
			return nil, fmt.Errorf("response status code: %d; unmarshal error: %s", resp.StatusCode, err)
		}

		return nil, fmt.Errorf("%s: %s", code, message)
	}

	cmd, err := flowstate.UnmarshalCommand(b)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return cmd, nil
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
