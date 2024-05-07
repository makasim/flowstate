package flowstate

type Watcher interface {
	Watch() chan []*TaskCtx
	Close()
}
