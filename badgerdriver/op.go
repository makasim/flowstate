package badgerdriver

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func setState(txn *badger.Txn, state flowstate.State) error {
	return setGOB(txn, stateKey(state), state)
}

func getState(txn *badger.Txn, stateID flowstate.StateID, stateRev int64) (flowstate.State, error) {
	state := flowstate.State{ID: stateID, Rev: stateRev}
	if err := getGOB(txn, stateKey(state), &state); err != nil {
		return flowstate.State{}, err
	}

	return state, nil
}

func stateKey(state flowstate.State) []byte {
	return []byte(fmt.Sprintf(`flowstate.states.%020d.%s`, state.Rev, state.ID))
}

func setGOB(txn *badger.Txn, k []byte, m any) error {
	var v bytes.Buffer
	if err := gob.NewEncoder(&v).Encode(m); err != nil {
		return err
	}

	if err := txn.Set(k, v.Bytes()); err != nil {
		return err
	}

	return nil
}

func getGOB(txn *badger.Txn, k []byte, m any) error {
	item, err := txn.Get(k)
	if err != nil {
		return err
	}

	return getItemGOB(item, m)
}

func getItemGOB(item *badger.Item, m any) error {
	if err := item.Value(func(val []byte) error {
		if err := gob.NewDecoder(bytes.NewBuffer(val)).Decode(m); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func setLatestRevIndex(txn *badger.Txn, state flowstate.State) error {
	return setInt64(txn, latestRevKey(state.ID), state.Rev)
}

func getLatestRevIndex(txn *badger.Txn, stateID flowstate.StateID) (int64, error) {
	rev, err := getInt64(txn, latestRevKey(stateID))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return rev, nil
}

func latestRevKey(stateID flowstate.StateID) []byte {
	return []byte(fmt.Sprintf(`flowstate.index.latest_rev.%s`, stateID))
}

func setInt64(txn *badger.Txn, key []byte, v int64) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))

	return txn.Set(key, b)
}

func getInt64(txn *badger.Txn, key []byte) (int64, error) {
	var val uint64
	var err error

	item, err := txn.Get(key)
	if err != nil {
		return 0, err
	}

	if err := item.Value(func(val0 []byte) error {
		defer func() {
			if recover() != nil {
				val = 0
				err = fmt.Errorf("value not uint64")
			}
		}()

		val = binary.BigEndian.Uint64(val0)
		return nil
	}); err != nil {
		return 0, err
	}

	return int64(val), err
}

func setLabelsIndex(txn *badger.Txn, state flowstate.State) error {
	for labelKey, labelVal := range state.Labels {
		if err := txn.Set(labelIndexKey(labelKey, labelVal, state.Rev), []byte(state.ID)); err != nil {
			return err
		}
	}
	return nil
}

func labelIndexKey(labelKey, labelVal string, stateRev int64) []byte {
	return []byte(fmt.Sprintf("%s%020d", labelIndexPrefix(labelKey, labelVal), stateRev))
}

func labelIndexPrefix(labelKey, labelVal string) []byte {
	return []byte(fmt.Sprintf("flowstate.index.labels.%s:%s.", labelKey, labelVal))
}

func setRevIndex(txn *badger.Txn, state flowstate.State) error {
	return txn.Set(revIndexKey(state.Rev), []byte(state.ID))
}

func revIndexKey(stateRev int64) []byte {
	return []byte(fmt.Sprintf("%s%020d", revIndexPrefix(), stateRev))
}

func revIndexPrefix() []byte {
	return []byte("flowstate.index.rev.")
}

func setCommittedAtIndex(txn *badger.Txn, state flowstate.State) error {
	return txn.Set(committedAtIndexKey(state.CommittedAt(), state.Rev), []byte(state.ID))
}

func committedAtIndexKey(committedAt time.Time, stateRev int64) []byte {
	return []byte(fmt.Sprintf("%s%020d.%020d", committedAtIndexPrefix(), committedAt.UnixMilli(), stateRev))
}

func committedAtIndexPrefix() []byte {
	return []byte("flowstate.index.committed_at.")
}

func dataBKey(data *flowstate.Data) []byte {
	return []byte(fmt.Sprintf(`flowstate.data.blob.%020d.%s`, data.Rev, data.ID))
}

func dataBinaryKey(data *flowstate.Data) []byte {
	return []byte(fmt.Sprintf(`flowstate.data.binary.%020d.%s`, data.Rev, data.ID))
}

func setData(txn *badger.Txn, data *flowstate.Data) error {
	if err := txn.Set(dataBKey(data), data.B); err != nil {
		return fmt.Errorf("set data.B: %w", err)
	}

	var dataBinary []byte
	if data.Binary {
		dataBinary = append(dataBinary, 1)
	}

	if err := txn.Set(dataBinaryKey(data), dataBinary); err != nil {
		return fmt.Errorf("set data.Binary: %w", err)
	}

	return nil
}

func getData(txn *badger.Txn, data *flowstate.Data) error {
	item, err := txn.Get(dataBKey(data))
	if err != nil {
		return fmt.Errorf("get data.Bd: %w", err)
	}
	data.B, err = item.ValueCopy(data.B)
	if err != nil {
		return fmt.Errorf("copy data.B: %w", err)
	}

	dataBinary, err := txn.Get(dataBinaryKey(data))
	if err != nil {
		return fmt.Errorf("get data.Binary: %w", err)
	}
	data.Binary = dataBinary.ValueSize() > 0

	return nil
}

func setDelayedState(txn *badger.Txn, delayedState flowstate.DelayedState) error {
	return setGOB(txn, delayedStateKey(delayedState.ExecuteAt.Unix(), delayedState.Offset), delayedState)
}

func getDelayedState(txn *badger.Txn, executeAt, offset int64) (flowstate.DelayedState, error) {
	delayedState := flowstate.DelayedState{}

	if err := getGOB(txn, delayedStateKey(executeAt, offset), delayedState); err != nil {
		return flowstate.DelayedState{}, fmt.Errorf("get delayed state: %w", err)
	}

	return delayedState, nil
}

func delayedStateKey(executeAt, offset int64) []byte {
	return []byte(fmt.Sprintf(`flowstate.delayed.states.%020d.%020d`, executeAt, offset))
}

func delayedStatePrefix(since time.Time) []byte {
	return []byte(fmt.Sprintf(`flowstate.delayed.states.%020d.`, since.Unix()))
}

func setDelayedOffsetIndex(txn *badger.Txn, delayedState flowstate.DelayedState) error {
	return txn.Set(
		delayedOffsetKey(delayedState.Offset),
		delayedStateKey(delayedState.ExecuteAt.Unix(), delayedState.Offset),
	)
}

func delayedOffsetKey(offset int64) []byte {
	return []byte(fmt.Sprintf(`flowstate.index.delayed_status_offset.%020d`, offset))
}

func delayedOffsetPrefix() []byte {
	return []byte(`flowstate.index.delayed_status_offset.`)
}
