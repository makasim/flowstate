package flowstate

import "time"

type WatchListener interface {
	Listen() <-chan State
	Close()
}

func Watch(labels map[string]string) *WatchCommand {
	return (&WatchCommand{}).WithORLabels(labels)
}

func DoWatch(e Engine, cmd *WatchCommand) (WatchListener, error) {
	if err := e.Do(cmd); err != nil {
		return nil, err
	}

	return cmd.Listener, nil
}

type WatchCommand struct {
	command
	SinceRev    int64
	SinceLatest bool
	SinceTime   time.Time
	Labels      []map[string]string

	Listener WatchListener
}

func (c *WatchCommand) WithSinceRev(rev int64) *WatchCommand {
	c.SinceLatest = false
	c.SinceRev = rev
	return c
}

func (c *WatchCommand) WithSinceLatest() *WatchCommand {
	c.SinceLatest = true
	c.SinceRev = 0
	return c
}

func (c *WatchCommand) WithSinceTime(since time.Time) *WatchCommand {
	c.SinceTime = since
	return c
}

func (c *WatchCommand) WithORLabels(labels map[string]string) *WatchCommand {
	if len(labels) == 0 {
		return c
	}

	c.Labels = append(c.Labels, labels)
	return c
}
