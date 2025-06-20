package badgerdriver

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

type badgerIterator struct {
	*badger.Iterator
	prefix []byte
}

type iterator interface {
	Valid() bool
	Current() flowstate.State
	Next()
	Close()
}

type labelsIterator struct {
	txn *badger.Txn
	its []*badgerIterator

	reverse bool
	valid   bool
	current flowstate.State
}

func newLabelsIterator(txn *badger.Txn, labels map[string]string, sinceRev int64, reverse bool) *labelsIterator {
	labelIt := &labelsIterator{
		txn:     txn,
		valid:   true,
		reverse: reverse,

		its: make([]*badgerIterator, 0, len(labels)),
	}

	if reverse && sinceRev == 0 {
		sinceRev = math.MaxInt64
	} else {
		sinceRev = sinceRev + 1
	}

	for labelKey, labelVal := range labels {
		if len(labelKey) == 0 || len(labelVal) == 0 {
			continue
		}

		prefix := labelIndexPrefix(labelKey, labelVal)
		it := &badgerIterator{
			prefix: prefix,
			Iterator: txn.NewIterator(badger.IteratorOptions{
				PrefetchSize: 100,
				Prefix:       prefix,
				Reverse:      reverse,
			}),
		}
		it.Seek(labelIt.seekPrefix(prefix, sinceRev))
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

func (lIt *labelsIterator) Next() {
	if !lIt.valid {
		return
	}

	for _, it := range lIt.its {
		it.Next()
		if !it.Valid() {
			lIt.Close()
			return
		}
	}

	lIt.seekState()
}

func (lIt *labelsIterator) Valid() bool {
	return lIt.valid
}

func (lIt *labelsIterator) Current() flowstate.State {
	return lIt.current
}

func (lIt *labelsIterator) Close() {
	lIt.valid = false
	for _, it := range lIt.its {
		it.Close()
	}
	lIt.its = nil
}

func (lIt *labelsIterator) toRev(it *badgerIterator) int64 {
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

func (lIt *labelsIterator) seekState() {
next:
	for lIt.valid {
		nextRev := int64(0)
		if lIt.reverse {
			nextRev = math.MaxInt64
		}
		for i := 0; i < len(lIt.its); i++ {
			if lIt.reverse {
				nextRev = min(lIt.toRev(lIt.its[i]), nextRev)
				continue
			}

			nextRev = max(lIt.toRev(lIt.its[i]), nextRev)
		}

		for _, it := range lIt.its {
			if !lIt.seekToRev(it, nextRev) {
				continue next
			}
		}

		var stateID flowstate.StateID
		if err := lIt.its[0].Item().Value(func(val []byte) error {
			stateID = flowstate.StateID(val)
			return nil
		}); err != nil {
			panic(fmt.Sprintf("BUG: cannot get state ID from key '%s': %s", lIt.its[0].Item().Key(), err))
		}

		state, err := getState(lIt.txn, stateID, nextRev)
		if err != nil {
			panic(fmt.Sprintf("BUG: cannot get state %s:%d: %s", stateID, nextRev, err))
		}

		lIt.current = state

		return
	}
}

func (lIt *labelsIterator) seekToRev(it *badgerIterator, toRev int64) bool {
	seekPrefix := lIt.seekPrefix(it.prefix, toRev)
	it.Seek(seekPrefix)

	if !it.Valid() {
		lIt.Close()
		return false
	}

	rev := lIt.toRev(it)

	if lIt.reverse {
		if rev < toRev {
			return false
		}
	} else {
		if rev > toRev {
			return false
		}
	}

	return true
}

func (lIt *labelsIterator) seekPrefix(prefix []byte, sinceRev int64) []byte {
	seekPrefix := append(prefix, fmt.Sprintf("%020d", sinceRev)...)
	if lIt.reverse {
		// TODO https://github.com/dgraph-io/badger/issues/347#issuecomment-350597302
		seekPrefix = append(seekPrefix, 0xFF)
	}

	return seekPrefix
}
