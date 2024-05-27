package flowstate

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

	Listener Watcher
}

func (cmd *WatchCommand) Prepare() error {
	return nil
}
