package pgdriver

import (
	"context"
	"log"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Watcher{}
var _ flowstate.WatchListener = &listener{}

type watcherQueries interface {
	GetStatesByLabels(ctx context.Context, tx conntx, orLabels []map[string]string, sinceRev int64, ss []flowstate.State) ([]flowstate.State, error)
}

type Watcher struct {
	conn conn
	q    watcherQueries
}

func NewWatcher(conn conn, q watcherQueries) *Watcher {
	d := &Watcher{
		conn: conn,
		q:    q,
	}

	return d
}

func (w *Watcher) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*flowstate.WatchCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	lis := &listener{
		conn: w.conn,
		q:    w.q,

		sinceRev:    cmd.SinceRev,
		sinceLatest: cmd.SinceLatest,
		sinceTime:   cmd.SinceTime,

		labels:  make([]map[string]string, 0),
		watchCh: make(chan flowstate.State, 1),
		closeCh: make(chan struct{}),
	}

	for _, labels := range cmd.Labels {
		lisLabels := make(map[string]string)
		for k, v := range labels {
			lisLabels[k] = v
		}
		lis.labels = append(lis.labels, lisLabels)
	}

	go lis.listen()

	cmd.Listener = lis

	return nil
}

func (w *Watcher) Init(_ flowstate.Engine) error {
	return nil
}

func (w *Watcher) Shutdown(_ context.Context) error {
	return nil
}

type listener struct {
	conn conn
	q    watcherQueries

	sinceRev    int64
	sinceLatest bool
	sinceTime   time.Time
	labels      []map[string]string

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
		lis.sinceRev = -1
	}

	t := time.NewTicker(time.Millisecond * 100)
	defer t.Stop()

	ss := make([]flowstate.State, 50)

skip:
	for {
		select {
		case <-t.C:
			ss = ss[0:50:50]
			for i := range ss {
				ss[i] = flowstate.State{}
			}

			var err error
			ss0, err := lis.q.GetStatesByLabels(context.Background(), lis.conn, lis.labels, lis.sinceRev, ss)
			if err != nil {
				log.Printf("ERROR: lstener: find states: %s", err)
				continue skip
			}
			if len(ss0) == 0 {
				continue skip
			}

			ss = ss0
		next:
			for i := range ss {
				nextSinceRev := ss[i].Rev
				select {
				case lis.watchCh <- ss[i]:
					lis.sinceRev = nextSinceRev
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
