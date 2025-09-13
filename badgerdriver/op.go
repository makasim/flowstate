package badgerdriver

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/makasim/flowstate"
)

func setState(txn *badger.Txn, state flowstate.State) error {
	return txn.Set(
		stateKey(state),
		flowstate.MarshalState(state, nil),
	)
}

func getState(txn *badger.Txn, stateID flowstate.StateID, stateRev int64) (flowstate.State, error) {
	state := flowstate.State{ID: stateID, Rev: stateRev}
	item, err := txn.Get(stateKey(state))
	if err != nil {
		return flowstate.State{}, err
	}

	if err := item.Value(func(val []byte) error {
		return flowstate.UnmarshalState(val, &state)
	}); err != nil {
		return flowstate.State{}, err
	}

	return state, nil
}

func stateKey(state flowstate.State) []byte {
	return []byte(fmt.Sprintf(`flowstate.states.%020d.%s`, state.Rev, state.ID))
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
	return txn.Set(committedAtIndexKey(state.CommittedAt, state.Rev), []byte(state.ID))
}

func committedAtIndexKey(committedAt time.Time, stateRev int64) []byte {
	return []byte(fmt.Sprintf("%s%020d.%020d", committedAtIndexPrefix(), committedAt.UnixMilli(), stateRev))
}

func committedAtIndexPrefix() []byte {
	return []byte("flowstate.index.committed_at.")
}

func dataBlobKey(data *flowstate.Data) []byte {
	return []byte(fmt.Sprintf(`flowstate.data.blob.%020d`, data.Rev))
}

func dataAnnotationsKey(data *flowstate.Data) []byte {
	return []byte(fmt.Sprintf(`flowstate.data.annotations.%020d`, data.Rev))
}

func setData(txn *badger.Txn, data *flowstate.Data) error {
	if err := txn.Set(dataBlobKey(data), data.Blob); err != nil {
		return fmt.Errorf("set data.Blob: %w", err)
	}

	if len(data.Annotations) > 0 {
		annotationsJSON, err := json.Marshal(data.Annotations)
		if err != nil {
			return fmt.Errorf("marshal data.Annotations: %w", err)
		}
		if err := txn.Set(dataAnnotationsKey(data), annotationsJSON); err != nil {
			return fmt.Errorf("set data.Annotations: %w", err)
		}
	}

	return nil
}

func getData(txn *badger.Txn, data *flowstate.Data) error {
	blobItem, err := txn.Get(dataBlobKey(data))
	if err != nil {
		return fmt.Errorf("get data.Blob: %w", err)
	}
	data.Blob, err = blobItem.ValueCopy(data.Blob)
	if err != nil {
		return fmt.Errorf("copy data.Blob: %w", err)
	}

	if annotationsItem, err := txn.Get(dataAnnotationsKey(data)); errors.Is(err, badger.ErrKeyNotFound) {
		// ok
	} else if err != nil {
		return fmt.Errorf("get data.Annotations: %w", err)
	} else {
		if err := annotationsItem.Value(func(val []byte) error {
			if len(val) == 0 {
				return nil
			}
			annotations := make(map[string]string)
			if err := json.Unmarshal(val, &annotations); err != nil {
				return fmt.Errorf("unmarshal annotations: %w", err)
			}

			data.Annotations = annotations
			return nil
		}); err != nil {
			return fmt.Errorf("copy data.Annotations: %w", err)
		}
	}

	return nil
}

func setDelayedState(txn *badger.Txn, delayedState flowstate.DelayedState) error {
	return txn.Set(
		delayedStateKey(delayedState.ExecuteAt.Unix(), delayedState.Offset),
		flowstate.MarshalDelayedState(delayedState, nil),
	)
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
