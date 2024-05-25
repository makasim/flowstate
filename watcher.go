package flowstate

type Watcher interface {
	Watch() <-chan *StateCtx
	Close()
}
