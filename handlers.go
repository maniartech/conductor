package choreo

type ActivityHandler func(any) bool

type ErrorHandler func(string, any, error)

// ChoreographyHandler provides a signature validation for
// future function.
//
// Example:
//
//	func Process(p *Choreography, ...v interface()) {
//	  processId := v.(int)
//	  result, err := SendRequest(processId)
//	  // When finished processing, call Done by passing
//	  // result and error details
//	  p.Done(result, err)
//	}
type ChoreographyHandler func(*Choreography, ...interface{})

// ThenHandler is a callback which will be
// invoked when the associated future has finished.
//
// Example
//
//	p := async.Go(process, 1)
//	p.Then(func(v interface{}, e error) {
//	  print("The future has just finished!")
//	})
type ThenHandler func(interface{}, error)
