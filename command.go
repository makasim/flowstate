package flowstate

type Command interface {
	Prepare() error
}
