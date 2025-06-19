package badgerdriver

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

type iterator struct {
	*badger.Iterator
	prefix []byte
}

type andLabelIterator struct {
	txn *badger.Txn
	its []*iterator

	valid   bool
	current flowstate.State
}

func newAndLabelIterator(txn *badger.Txn, labels map[string]string, sinceRev int64) *andLabelIterator {
	labelIt := &andLabelIterator{
		txn:   txn,
		valid: true,

		its: make([]*iterator, 0, len(labels)),
	}

	for labelKey, labelVal := range labels {
		if len(labelKey) == 0 || len(labelVal) == 0 {
			continue
		}

		prefix := stateLabelPrefix(labelKey, labelVal)
		it := &iterator{
			prefix: prefix,
			Iterator: txn.NewIterator(badger.IteratorOptions{
				PrefetchSize: 100,
				Prefix:       prefix,
			}),
		}
		seekPrefix := append(prefix, fmt.Sprintf("%020d", sinceRev)...)
		it.Seek(seekPrefix)

		if !it.Valid() {
			it.Close()
			labelIt.Close()
		}

		labelIt.its = append(labelIt.its, it)
	}

	if len(labelIt.its) == 0 {
		labelIt.Close()
		return labelIt
	}

	labelIt.seekState()

	return labelIt
}

func (labelIt *andLabelIterator) Next() {
	if !labelIt.valid {
		return
	}

	for _, it := range labelIt.its {
		it.Next()
		if !it.Valid() {
			labelIt.Close()
			return
		}
	}

	labelIt.seekState()
}

func (labelIt *andLabelIterator) Valid() bool {
	return labelIt.valid
}

func (labelIt *andLabelIterator) Current() flowstate.State {
	return labelIt.current
}

func (labelIt *andLabelIterator) Close() {
	labelIt.valid = false
	for _, it := range labelIt.its {
		it.Close()
	}
	labelIt.its = nil
}

func (labelIt *andLabelIterator) toRev(it *iterator) int64 {
	after, found := bytes.CutPrefix(it.Item().Key(), it.prefix)
	if !found {
		panic(fmt.Sprintf("BUG: key '%s' does not start with prefix '%s'", it.Item().Key(), string(it.prefix)))
	}

	rev, err := strconv.ParseInt(string(after), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("BUG: cannot parse revision from key %s: %s", it.Item().Key(), err.Error()))
	}

	return rev
}

func (labelIt *andLabelIterator) seekState() {
next:
	for labelIt.valid {
		nextRev := int64(0)
		for i := 0; i < len(labelIt.its); i++ {
			nextRev = max(labelIt.toRev(labelIt.its[i]), nextRev)
		}

		for _, it := range labelIt.its {
			if !labelIt.seekToRev(it, nextRev) {
				continue next
			}
		}

		var stateID flowstate.StateID
		if err := labelIt.its[0].Item().Value(func(val []byte) error {
			stateID = flowstate.StateID(val)
			return nil
		}); err != nil {
			panic(fmt.Sprintf("BUG: cannot get state ID from key '%s': %s", labelIt.its[0].Item().Key(), err))
		}

		state, err := getState(labelIt.txn, stateID, nextRev)
		if err != nil {
			panic(fmt.Sprintf("BUG: cannot get state %s:%d: %s", stateID, nextRev, err))
		}

		labelIt.current = state

		return
	}
}

func (labelIt *andLabelIterator) seekToRev(it *iterator, toRev int64) bool {
	seekPrefix := append(it.prefix, fmt.Sprintf("%020d", toRev)...)
	it.Seek(seekPrefix)

	if !it.Valid() {
		labelIt.Close()
		return false
	}

	rev := labelIt.toRev(it)
	if rev > toRev {
		return false
	}

	return true
}
