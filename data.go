package flowstate

import (
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
	jsonD := &jsonData{}
	jsonD.fromData(d)
	return json.Marshal(jsonD)
}

func (d *Data) UnmarshalJSON(data []byte) error {
	jsonD := &jsonData{}
	if err := json.Unmarshal(data, jsonD); err != nil {
		return err
	}

	return jsonD.toData(d)
}
