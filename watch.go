package flowstate

type WatchListener interface {
	Watch() <-chan *StateCtx
	Close()
}

func Watch(sinceRev int64, labels map[string]string) *WatchCommand {
	return &WatchCommand{
		SinceRev: sinceRev,
		Labels:   labels,
	}
}

type WatchCommand struct {
	SinceRev    int64
	SinceLatest bool
	Labels      map[string]string

	Listener WatchListener
}

// A driver must implement a command doer.
