package badgerdriver

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"

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

func setLatestStateRev(txn *badger.Txn, state flowstate.State) error {
	return setInt64(txn, latestStateRevKey(state), state.Rev)
}

func getLatestStateRev(txn *badger.Txn, state flowstate.State) (int64, error) {
	rev, err := getInt64(txn, latestStateRevKey(state))
	if errors.Is(err, badger.ErrKeyNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return rev, nil
}

func setStateLabel(txn *badger.Txn, stateID flowstate.StateID, stateRev int64, labelKey, labelVal string) error {
	return txn.Set(
		[]byte(fmt.Sprintf("flowstate.labels_index.%d.%s:%s", stateRev, labelKey, labelVal)),
		[]byte(fmt.Sprintf("%s:%d", stateID, stateRev)),
	)
}

//func getStateLabel(txn *badger.Txn, labelKey, labelVal string) (flowstate.StateID, int64, error) {
//	item, err := txn.Get([]byte(fmt.Sprintf("flowstate.labels_index.%d.%s:%s", stateRev, labelKey, labelVal)))
//	if err != nil {
//		return "", 0, err
//	}
//
//	val0, err := item.ValueCopy(nil)
//	if err != nil {
//		return "", 0, err
//	}
//	parts := bytes.Split(val0, []byte(":"))
//	stateID := flowstate.StateID(parts[0])
//	stateRev, err := strconv.ParseInt(string(parts[1]), 10, 64)
//	if err != nil {
//		return "", 0, err
//	}
//
//	return stateID, stateRev, nil
//}

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

func stateKey(state flowstate.State) []byte {
	return []byte(fmt.Sprintf(`flowstate.states.%d.%s`, state.Rev, state.ID))
}

func latestStateRevKey(state flowstate.State) []byte {
	return []byte(fmt.Sprintf(`flowstate.latest_states.%s`, state.ID))
}
