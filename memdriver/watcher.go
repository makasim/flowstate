package memdriver

import (
	"context"
	"time"

	"github.com/makasim/flowstate"
)

var _ flowstate.Doer = &Watcher{}
var _ flowstate.WatchListener = &listener{}

type Watcher struct {
	l *Log
	e *flowstate.Engine
}

func NewWatcher(l *Log) *Watcher {
	d := &Watcher{
		l: l,
	}

	return d
}

func (w *Watcher) Do(cmd0 flowstate.Command) error {
	cmd, ok := cmd0.(*flowstate.WatchCommand)
	if !ok {
		return flowstate.ErrCommandNotSupported
	}

	lis := &listener{
		l: w.l,

		sinceRev:    cmd.SinceRev,
		sinceLatest: cmd.SinceLatest,
		sinceTime:   cmd.SinceTime,
		labels:      make(map[string]string),

		watchCh:  make(chan flowstate.State, 1),
		changeCh: make(chan int64, 1),
		closeCh:  make(chan struct{}),
	}

	for k, v := range cmd.Labels {
		lis.labels[k] = v
	}

	lis.change(cmd.SinceRev)

	if err := w.l.SubscribeCommit(lis.changeCh); err != nil {
		return err
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
	l *Log

	sinceRev    int64
	sinceLatest bool
	sinceTime   time.Time
	labels      map[string]string
	limit       int

	watchCh  chan flowstate.State
	changeCh chan int64

	closeCh chan struct{}
}

func (lis *listener) Listen() <-chan flowstate.State {
	return lis.watchCh
}

func (lis *listener) Close() {
	close(lis.closeCh)
}

func (lis *listener) change(rev int64) {
	select {
	case lis.changeCh <- rev:
	case <-lis.changeCh:
		lis.changeCh <- rev
	}
}

func (lis *listener) listen() {
	var states []*flowstate.StateCtx

	if lis.sinceLatest {
		lis.l.Lock()
		_, sinceRev := lis.l.GetLatestByLabels(lis.labels)
		lis.sinceRev = sinceRev - 1
		lis.l.Unlock()
	}

skip:
	for {
		select {
		case <-lis.changeCh:
			for {
				lis.l.Lock()
				states, lis.sinceRev = lis.l.Entries(lis.sinceRev, 10)
				lis.l.Unlock()

				if len(states) == 0 {
					continue skip
				}

			next:
				for _, s := range states {
					if !lis.sinceTime.IsZero() && lis.sinceTime.UnixMilli() < s.Committed.CommittedAtUnixMilli {
						continue next
					}
					for k, v := range lis.labels {
						if s.Committed.Labels[k] != v {
							continue next
						}
					}

					select {
					case lis.watchCh <- s.Committed:
						continue next
					case <-lis.closeCh:
						return
					}
				}
			}
		case <-lis.closeCh:
			lis.l.UnsubscribeCommit(lis.changeCh)
			return
		}
	}
}
