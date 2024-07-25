package sqlitedriver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Watcher{}
var _ flowstate.WatchListener = &listener{}

type Watcher struct {
	db *sql.DB
	e  *flowstate.Engine
}

func NewWatcher(db *sql.DB) *Watcher {
	d := &Watcher{
		db: db,
	}

	return d
}

func (w *Watcher) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*flowstate.WatchCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	lis := &listener{
		db: w.db,

		sinceRev:    cmd.SinceRev,
		sinceLatest: cmd.SinceLatest,
		sinceTime:   cmd.SinceTime,

		labels:  make(map[string]string),
		watchCh: make(chan flowstate.State, 1),
		closeCh: make(chan struct{}),
	}

	for k, v := range cmd.Labels {
		lis.labels[k] = v
	}

	go lis.listen()

	cmd.Listener = lis

	return nil
}

func (w *Watcher) Init(e *flowstate.Engine) error {
	w.e = e
	return nil
}

func (w *Watcher) Shutdown(_ context.Context) error {
	return nil
}

type listener struct {
	db *sql.DB

	sinceRev    int64
	sinceLatest bool
	sinceTime   time.Time
	labels      map[string]string

	watchCh chan flowstate.State

	closeCh chan struct{}
}

func (lis *listener) Listen() <-chan flowstate.State {
	return lis.watchCh
}

func (lis *listener) Close() {
	close(lis.closeCh)
}

func (lis *listener) listen() {
	if lis.sinceLatest {
		sinceLatest, err := lis.findSinceLatest()
		if err != nil {
			log.Printf("ERROR: listner: find since latest: %s", err)
			return
		}

		lis.sinceRev = sinceLatest
	}

	var states []flowstate.State

	t := time.NewTicker(time.Millisecond * 100)
	defer t.Stop()

skip:
	for {

		select {
		case <-t.C:
			var err error
			states, err = lis.findStates()
			if err != nil {
				log.Printf("ERROR: lstener: find states: %s", err)
				continue skip
			}

			if len(states) == 0 {
				continue skip
			}

		next:
			for _, s := range states {
				select {
				case lis.watchCh <- s:
					lis.sinceRev = s.Rev
					continue next
				case <-lis.closeCh:
					return
				}
			}
		case <-lis.closeCh:
			return
		}
	}
}

func (lis *listener) findStates() ([]flowstate.State, error) {
	args := make([]interface{}, 0, len(lis.labels)+2)

	var labelsWhere string
	for k, v := range lis.labels {
		if labelsWhere != "" {
			labelsWhere += " AND "
		}

		// TODO: somewhat ? does not work inside josn_extract, sanitize k to prevent sql injection
		labelsWhere += `json_extract(state, '$.labels.` + k + `') = ?`
		args = append(args, v)
	}
	if labelsWhere == "" {
		labelsWhere = " TRUE "
	}

	if !lis.sinceTime.IsZero() {
		labelsWhere += " AND json_extract(state, '$.committed_at_unix_milli') > ?"
		args = append(args, lis.sinceTime.UnixMilli())
	}

	args = append(args, lis.sinceRev, 10)

	q := fmt.Sprintf(`
SELECT 
    state 
FROM 
    flowstate_state_log 
WHERE 
    %s AND rev > ?
ORDER BY rev 
LIMIT ?`, labelsWhere)

	rows, err := lis.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stateJSON []byte

	var states []flowstate.State
	for rows.Next() {
		stateJSON = stateJSON[:0]

		if err := rows.Scan(&stateJSON); err != nil {
			return nil, err
		}

		var state flowstate.State
		if err := json.Unmarshal(stateJSON, &state); err != nil {
			return nil, err
		}

		states = append(states, state)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return states, nil
}

func (lis *listener) findSinceLatest() (int64, error) {
	args := make([]interface{}, 0, len(lis.labels)+2)

	var labelsWhere string
	for k, v := range lis.labels {
		if labelsWhere != "" {
			labelsWhere += " AND "
		}

		// TODO: somewhat ? does not work inside josn_extract, sanitize k to prevent sql injection
		labelsWhere += `json_extract(state, '$.labels.` + k + `') = ?`
		args = append(args, v)
	}

	args = append(args, lis.sinceRev, 10)

	if labelsWhere == "" {
		labelsWhere = " TRUE "
	}

	q := fmt.Sprintf(`
SELECT 
    rev 
FROM 
    flowstate_state_log 
WHERE 
    %s
ORDER BY rev DESC 
LIMIT 1`, labelsWhere)

	var sinceRev int64
	if err := lis.db.QueryRow(q, args...).Scan(&sinceRev); err != nil {
		return 0, err
	}

	sinceRev -= 1

	return sinceRev, nil
}
