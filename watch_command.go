package flowstate

func Watch(since int64, labels map[string]string) *WatchCommand {
	return &WatchCommand{
		Since:  since,
		Labels: labels,
	}

}

type WatchCommand struct {
	Since  int64
	Labels map[string]string

	Watcher Watcher
}

func (cmd *WatchCommand) Prepare() error {
	return nil
}
