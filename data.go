package flowstate

type DataID string

type Data struct {
	noCopy

	ID     DataID
	Rev    int64
	Binary bool

	B []byte
}

func (d *Data) CopyTo(to *Data) *Data {
	to.ID = d.ID
	to.Rev = d.Rev
	to.Binary = d.Binary
	to.B = append(to.B[:0], d.B...)

	return to
}
