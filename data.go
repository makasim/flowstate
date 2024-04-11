package flowstate

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type DataID string

type Data struct {
	ID  DataID `json:"id"`
	Rev int64  `json:"rev"`

	Bytes []byte `json:"bytes"`
}

func (d *Data) Set(path string, value interface{}) error {
	if len(d.Bytes) == 0 {
		d.Bytes = append(d.Bytes, `{}`...)
	}

	res, err := sjson.SetBytes(d.Bytes, path, value)
	if err != nil {
		return err
	}

	d.Bytes = res
	return nil
}

func (d *Data) Get(path string) (interface{}, bool) {
	if len(d.Bytes) == 0 {
		return nil, false
	}

	res := gjson.GetBytes(d.Bytes, path)
	return res.Value(), res.Exists()
}
