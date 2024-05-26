package memdriver

import (
	"github.com/makasim/flowstate"
)

type Watcher struct {
	l *Log
}

func NewWatcher(l *Log) *Watcher {
	d := &Watcher{
		l: l,
	}

	return d
}

func (d *Watcher) Do(cmd0 flowstate.Command) (*flowstate.StateCtx, error) {
	cmd, ok := cmd0.(*flowstate.WatchCommand)
	if !ok {
		return nil, flowstate.ErrCommandNotSupported
	}

	lis := &listener{
		l: d.l,

		sinceRev:    cmd.SinceRev,
		sinceLatest: cmd.SinceLatest,
		// todo: copy labels
		labels: cmd.Labels,

		watchCh:  make(chan *flowstate.StateCtx, 1),
		changeCh: make(chan int64, 1),
		closeCh:  make(chan struct{}),
	}

	lis.Change(cmd.SinceRev)

	if err := d.l.SubscribeCommit(lis.changeCh); err != nil {
		return nil, err
	}

	go lis.listen()

	cmd.Listener = lis

	return nil, nil
}

type listener struct {
	l *Log

	sinceRev    int64
	sinceLatest bool

	labels   map[string]string
	watchCh  chan *flowstate.StateCtx
	changeCh chan int64

	closeCh chan struct{}
}

func (lis *listener) Watch() <-chan *flowstate.StateCtx {
	return lis.watchCh
}

func (lis *listener) Close() {
	close(lis.closeCh)
}

func (lis *listener) Change(rev int64) {
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
		_, sinceRev := lis.l.LatestByLabels(lis.labels)
		lis.sinceRev = sinceRev - 1
		lis.l.Unlock()
	}

skip:
	for {
		select {
		case <-lis.changeCh:
			lis.l.Lock()
			states, lis.sinceRev = lis.l.Entries(lis.sinceRev, 10)
			lis.l.Unlock()

			if len(states) == 0 {
				continue skip
			}

		next:
			for _, t := range states {
				for k, v := range lis.labels {
					if t.Committed.Labels[k] != v {
						continue next
					}
				}

				select {
				case lis.watchCh <- t:
					continue next
				case <-lis.closeCh:
					return
				}
			}
		case <-lis.closeCh:
			lis.l.UnsubscribeCommit(lis.changeCh)
			return
		}
	}
}
