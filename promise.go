package async

type promise interface {
	Start()

	OnReady(func(interface{}, error))

	Done(...interface{})

	Await()

	Err() error

	// Resut
	Result() interface{}

	// Parent returns the parent promise
	Parent()

	Children() []Promise

	Status() string // not-started, started, finished
}
