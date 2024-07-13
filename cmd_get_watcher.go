package flowstate

func GetWatcher(sinceRev int64, labels map[string]string) *GetWatcherCommand {
	return &GetWatcherCommand{
		SinceRev: sinceRev,
		Labels:   labels,
	}
}

type GetWatcherCommand struct {
	command
	SinceRev    int64
	SinceLatest bool
	Labels      map[string]string

	Watcher Watcher
}
