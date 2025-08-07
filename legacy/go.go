package orchestrator

// Func creates a new future which provides an easy await mechanism.
// It can be started either by calling the `Start` or `Await` method.
//
//    func(fn OrchestrationHandler, args ...interface{}) *Promise
//
// Example: Immediate start and await
//
//    // Starts a new process and awaits for it to finish.
//    v, err := async.Func(process, 1).Await()
//    if err != nil {
//      println("An error occurred while processing the future.")
//    }
//    print(v) // Print the resulted value
//
// Example: Delayed start
//
//    // Create a new future
//    p := async.Func(process, 1)
//    p.Then(func (v interface{}, e error) {
//      println("The process 1 finished.")
//    })
//
func Func(fn OrchestrationHandler, args ...interface{}) *Orchestration {
	return create(fn, args...)
}

// Async creates a new future from a list of futures and runs them in parallel goroutines.
// It returns the pointer to the newly created future.
//
//    //
//    func(futures ...*Orchestration) *Orchestration
//
// Example: (1)
//
//    async.Async(
//      async.Func(process, 1),
//      async.Func(sendEmail, 2)
//    ).Await()
//
func Async(futures ...*Orchestration) *Orchestration {
	return createBatch(false, futures...)
}

// Sync creates a new future from a list of futures and runs them in sequential
// goroutines! It returns the pointer to the newly created future.
//
// It accepts the following function signatures.
//
//     1) func(futures ...*Orchestration) *Orchestration
//     2) func(handlerFn OrchestrationHandler, args ...interface{}) *Promise
//
// Example: (1)
//  async.Sync(async.Func(process, 1), async.Func(sendEmail, 2))
//  async.Sync(async.NewOrchestration(process, 1), async.NewPromise(process, 2))
//
// Example: (2)
//   async.Func(process, 1) // Just runs the goroutine
//   async.Func(process, 2).Await() // Runs the goroutine and awaits for it to finish.
//
func Sync(futures ...*Orchestration) *Orchestration {
	return createBatch(true, futures...)
}
