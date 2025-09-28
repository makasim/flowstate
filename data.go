package flowstate

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cespare/xxhash/v2"
)

type Data struct {
	noCopy

	Rev         int64             `json:"-"`
	Annotations map[string]string `json:"-"`
	Blob        []byte            `json:"-"`
}

func (d *Data) SetAnnotation(name, value string) {
	if d.Annotations == nil {
		d.Annotations = make(map[string]string)
	}
	d.Annotations[name] = value
}

func (d *Data) isDirty() bool {
	if d.Rev == 0 {
		return true
	}
	if d.Annotations["checksum/xxhash64"] != "" {
		storedHash, err := strconv.ParseUint(d.Annotations["checksum/xxhash64"], 10, 64)
		if err != nil {
			return true
		}

		return storedHash != xxhash.Sum64(d.Blob)
	}

	return false
}

func (d *Data) SetBinary(v bool) {
	if v {
		d.SetAnnotation("binary", "true")
	} else {
		delete(d.Annotations, "binary")
	}
}

func (d *Data) IsBinary() bool {
	return d.Annotations["binary"] == "true"
}

func (d *Data) checksum() {
	d.SetAnnotation("checksum/xxhash64", strconv.FormatUint(xxhash.Sum64(d.Blob), 10))
}

func (d *Data) CopyTo(to *Data) *Data {
	to.Rev = d.Rev
	to.Blob = append(to.Blob[:0], d.Blob...)

	if len(d.Annotations) > 0 {
		if to.Annotations == nil {
			to.Annotations = make(map[string]string)
		}

		for k, v := range d.Annotations {
			to.Annotations[k] = v
		}
	}

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

func referenceData(stateCtx *StateCtx, alias string, dRev int64) {
	stateCtx.Current.SetAnnotation(
		attachDataAnnotation(alias),
		strconv.FormatInt(dRev, 10),
	)
}

func dereferenceData(stateCtx *StateCtx, alias string) (int64, error) {
	if alias == "" {
		return 0, fmt.Errorf("alias is empty")
	}

	annotKey := attachDataAnnotation(alias)
	revStr := stateCtx.Current.Annotations[annotKey]
	if revStr == "" {
		return 0, fmt.Errorf("annotation %q is not set", annotKey)
	}

	rev, err := strconv.ParseInt(revStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("annotation %q contains invalid data revision; got %q: %w", annotKey, revStr, err)
	}

	return rev, nil
}

func attachDataAnnotation(alias string) string {
	return "flowstate.data." + string(alias)
}
