package badgerdriver

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/makasim/flowstate"

	"github.com/dgraph-io/badger/v4"
)

func newOrLabelsIterator(txn *badger.Txn, orLabels []map[string]string, sinceRev int64, reverse bool) iterator {
	if len(orLabels) == 0 || len(orLabels[0]) == 0 {
		return newRevIterator(txn, sinceRev, reverse)
	}

	var its []iterator
	for _, labels := range orLabels {
		its = append(its, newLabelsIterator(txn, labels, sinceRev, reverse))
	}

	return newOrIterator(its, reverse)
}

func newOrIterator(its []iterator, reverse bool) *orIterator {
	orIt := &orIterator{
		valid:   true,
		reverse: reverse,
		its:     its,
	}

	orIt.seekState()

	return orIt
}

type orIterator struct {
	valid   bool
	reverse bool
	current flowstate.State
	its     []iterator
}

func (orIt *orIterator) Next() {
	if !orIt.valid {
		return
	}

	for _, it := range orIt.its {
		if it.Current().Rev == orIt.current.Rev {
			it.Next()
			break
		}
	}

	orIt.seekState()
}

func (orIt *orIterator) Valid() bool {
	return orIt.valid
}

func (orIt *orIterator) Current() flowstate.State {
	return orIt.current
}

func (orIt *orIterator) Close() {
	orIt.valid = false
	for _, it := range orIt.its {
		it.Close()
	}
	orIt.its = nil
}

func (orIt *orIterator) seekState() {
	var nextState flowstate.State
	for _, it := range orIt.its {
		if !it.Valid() {
			continue
		}

		state := it.Current()
		if nextState.Rev == 0 {
			nextState = state
			continue
		}
		if orIt.reverse {
			if state.Rev > nextState.Rev {
				nextState = state
			}
		} else {
			if state.Rev < nextState.Rev {
				nextState = state
			}
		}

	}

	its := make([]iterator, 0, len(orIt.its))
	for _, it := range orIt.its {
		if !it.Valid() {
			it.Close()
			continue
		}

		its = append(its, it)
	}

	if len(its) == 0 {
		orIt.Close()
		return
	}

	orIt.current = nextState
}

func newRevIterator(txn *badger.Txn, sinceRev int64, reverse bool) *revIterator {
	prefix := revIndexPrefix()

	if reverse && sinceRev == 0 {
		sinceRev = math.MaxInt64
	}

	rIt := &revIterator{
		reverse: true,
		txn:     txn,
		Iterator: txn.NewIterator(badger.IteratorOptions{
			PrefetchSize: 100,
			Prefix:       prefix,
			Reverse:      reverse,
		}),
	}
	rIt.Seek(rIt.seekPrefix(prefix, sinceRev))

	if rIt.Valid() {
		rIt.seekState()
	}

	return rIt
}

type revIterator struct {
	*badger.Iterator
	txn     *badger.Txn
	current flowstate.State
	reverse bool
}

func (rIt *revIterator) Next() {
	rIt.Iterator.Next()
	if !rIt.Valid() {
		return
	}

	rIt.seekState()
}

func (rIt *revIterator) Current() flowstate.State {
	return rIt.current
}

func (rIt *revIterator) seekState() {
	prefix := revIndexPrefix()
	key := rIt.Item().Key()

	after, found := bytes.CutPrefix(key, prefix)
	if !found {
		panic(fmt.Sprintf("BUG: key must start with prefix %s; got %s", prefix, key))
	}

	rev, err := strconv.ParseInt(string(after), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("BUG: cannot parse revision from key '%s': %s", key, err))
	}

	var stateID flowstate.StateID
	if err := rIt.Item().Value(func(val []byte) error {
		stateID = flowstate.StateID(val)
		return nil
	}); err != nil {
		panic(fmt.Sprintf("BUG: cannot get state ID from key '%s': %s", rIt.Item().Key(), err))
	}

	state, err := getState(rIt.txn, stateID, rev)
	if err != nil {
		panic(fmt.Sprintf("BUG: cannot get state %s:%d: %s", stateID, rev, err))
	}
	rIt.current = state
}

func (rIt *revIterator) seekPrefix(prefix []byte, sinceRev int64) []byte {
	seekPrefix := append(prefix, fmt.Sprintf("%020d", sinceRev)...)
	if rIt.reverse {
		// TODO https://github.com/dgraph-io/badger/issues/347#issuecomment-350597302
		seekPrefix = append(seekPrefix, 0xFF)
	}

	return seekPrefix
}

func newCommittedAtIterator(txn *badger.Txn, sinceTime time.Time, reverse bool) *committedAtIterator {
	prefix := committedAtIndexPrefix()

	if reverse {
		if sinceTime.IsZero() {
			sinceTime = time.Unix(math.MaxInt64, 0)
		}
	}

	caIt := &committedAtIterator{
		reverse: true,
		txn:     txn,
		Iterator: txn.NewIterator(badger.IteratorOptions{
			PrefetchSize: 100,
			Prefix:       prefix,
			Reverse:      reverse,
		}),
	}
	caIt.Seek(caIt.seekPrefix(prefix, sinceTime))

	if caIt.Valid() {
		caIt.seekState()
	}

	return caIt
}

type committedAtIterator struct {
	*badger.Iterator
	txn     *badger.Txn
	current flowstate.State
	reverse bool
}

func (caIt *committedAtIterator) Next() {
	caIt.Iterator.Next()
	if !caIt.Valid() {
		return
	}

	caIt.seekState()
}

func (caIt *committedAtIterator) Current() flowstate.State {
	return caIt.current
}

func (caIt *committedAtIterator) seekState() {
	prefix := committedAtIndexPrefix()
	key := caIt.Item().Key()

	idx := bytes.LastIndexByte(key, '.')
	if idx == -1 {
		panic(fmt.Sprintf("BUG: key must start with the prefix %s; got %s", prefix, string(key)))
	}

	rev, err := strconv.ParseInt(string(key[idx+1:]), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("BUG: cannot parse revision from key '%s': %s", key, err))
	}

	var stateID flowstate.StateID
	if err := caIt.Item().Value(func(val []byte) error {
		stateID = flowstate.StateID(val)
		return nil
	}); err != nil {
		panic(fmt.Sprintf("BUG: cannot get state ID from key '%s': %s", caIt.Item().Key(), err))
	}

	state, err := getState(caIt.txn, stateID, rev)
	if err != nil {
		panic(fmt.Sprintf("BUG: cannot get state %s:%d: %s", stateID, rev, err))
	}

	caIt.current = state
}

func (caIt *committedAtIterator) seekPrefix(prefix []byte, sinceTime time.Time) []byte {
	seekPrefix := append(prefix, fmt.Sprintf("%020d", sinceTime.UnixMilli())...)
	if caIt.reverse {
		// TODO https://github.com/dgraph-io/badger/issues/347#issuecomment-350597302
		seekPrefix = append(seekPrefix, 0xFF)
	}

	return seekPrefix
}
