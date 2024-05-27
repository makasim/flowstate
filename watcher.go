package flowstate

type WatchListener interface {
	Watch() <-chan *StateCtx
	Close()
}
