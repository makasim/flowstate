package flowstate

type Watcher interface {
	Watch() <-chan State
	Close()
}
