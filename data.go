package flowstate

import (
	"encoding/base64"
	"encoding/json"
)

type DataID string

type Data struct {
	noCopy

	ID     DataID `json:"-"`
	Rev    int64  `json:"-"`
	Binary bool   `json:"-"`

	B []byte `json:"-"`
}

func (d *Data) CopyTo(to *Data) *Data {
	to.ID = d.ID
	to.Rev = d.Rev
	to.Binary = d.Binary
	to.B = append(to.B[:0], d.B...)

	return to
}

func (d *Data) MarshalJSON() ([]byte, error) {
	var b string
	if d.Binary {
		b = base64.StdEncoding.EncodeToString(d.B)
	} else {
		b = string(d.B)
	}

	return json.Marshal(&struct {
		ID     DataID `json:"id,omitempty"`
		Rev    int64  `json:"rev,omitempty"`
		Binary bool   `json:"binary,omitempty"`

		B string `json:"b,omitempty"`
	}{
		ID:     d.ID,
		Rev:    d.Rev,
		Binary: d.Binary,
		B:      b,
	})
}

func (d *Data) UnmarshalJSON(data []byte) error {
	d0 := &struct {
		ID     DataID `json:"id,omitempty"`
		Rev    int64  `json:"rev,omitempty"`
		Binary bool   `json:"binary,omitempty"`

		B string `json:"b,omitempty"`
	}{}
	if err := json.Unmarshal(data, &d0); err != nil {
		return err
	}

	d.ID = d0.ID
	d.Rev = d0.Rev
	d.Binary = d0.Binary

	if d0.B == "" {
		return nil
	}

	if d0.Binary {
		b, err := base64.StdEncoding.DecodeString(d0.B)
		if err != nil {
			return err
		}
		d.B = b
	} else {
		d.B = []byte(d0.B)
	}

	return nil
}
