package conductor

// Like orchestra conductor, this package provides a way to
// orchestrate the execution of multiple functions in a senchronous or
// asynchronus manner. It also provides a way to handle errors and
// provide a way to handle the result of the function execution. It
// manages the root context and provides a way to cancel the execution.
// It also provides the read only access to the result of the functions.
// It stores the result of the function execution in a thread safe way.
// These results can be accessed by the functions which are executed
// after the function which produced the result. It also provides a way
// to cancel the entire execution or a specific function execution. While
// execution, each function receives the conductor context which provides
// the access to the result of the functions which are executed before it.
type conductor struct {
	future *Future

	Results map[string]interface{}
}

// New creates a new conductor instance.
func New(future *Future) *conductor {
	return &conductor{
		future: future,
	}
}
