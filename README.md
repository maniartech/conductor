# Easy go routines synchronization and orchestration!  [![Build Status](https://travis-ci.com/maniartech/async.svg?branch=master)](https://travis-ci.com/maniartech/async)

The `github.com/maniartech/async` is a tiny go library that aims to simplify the goroutine orchestration using easy to handle Async/Await pattern. This library provides a super-easy way to orchestrate the Goroutines using easily readable declarative syntax.

## Getting Started

Run the following command in your project to get the `async`.
```sh
go get github.com/maniartech/async
```


## Simple Async/Await
A simple async/await!
```go
package main

import "github.com/maniartech/async"


func main() {
  // Executes the async function in a new goroutine and
  // awaits for the result.
  result, err := async.Go(Process).Await()
  if err != nil {
    panic("An error occured while executing Process")
  }

  // Pass the result of previous goroutine to the next one!
  result, err = async.Go(Process2, result).Await()
  if err != nil {
    panic("An error occured while executing Process2")
  }

  println(p1.Result.Value, p2.Result.value)
}
```

## Commonly used functions



* `async.Go(FutureHandler, ...interfaces{}) *Promsie`

  Executes the function in a new goroutine and returns a future. The future can be awaited until the execution is finished and results are returned.

  **Example:**

  ```go
  // Return the pointer to Future
  future := async.Go(complexFunction)
  future.Await() // Awaits here!
  fmt.Printf("Result: %+v", future.Result)

  // Use await to return the results and error
  result, err := async.Go(complexFunction).Await()
  if err == nil {
    fmt.Printf("Result: %+v", future.Result)
  }
  ```

* `async.GoC(futures ...*Future) *Future`

  Executes the specified futures in the concurrent manner. Returns the future which can be used to await

* `async.GoQ(futures ...*Future) *Future`

  Executes the specified futures in the sequencial manner. Returns the future which can be used to await

## Complex Goroutine Orchestration

The following hypothetical example shows how a complex goroutines pipeline can be orchestrated using a simple structure!

```go
import "github.com/maniartech/async"


// HandleResource processes the various activities
// on the specified resource. All these activities
// are executed using their goroutines and
// in an orchestrated manner.
//
// This orchestration provides the concurrent, faster yet
// controlled execution of various activities.
//
//             |-----Go---------------------|     |-----Go----|
//             |                            |     |           |
// ----GoQ----GoC----GoQ->>-Go->>-Go->>-Go--|----GoC----Go----|----Await----
//             |                            |     |           |
//             |      |-----Go----|         |     |-----Go----|
//             |      |           |         |
//             |-----GoC----Go----|---------|
//                    |           |
//                    |-----Go----|
//

func HandleResource(resourceId int) {
  async.GoQ(
    async.GoC( // GoC: Concurrent execution
      async.Go(keepInfraReady),
      async.GoQ( // GoQ: Sequential execution
        async.Go(fetchResource, resourceId),
        async.Go(processResource),
        async.Go(submitResource),
      ),
      async.GoC(
        async.Go(prepareDependencyA)
        async.Go(prepareDependencyB)
        async.Go(prepareDependencyC)
      )
    ),
    async.GoC(
      async.Go(postToSocialMedia),
      async.Go(sendNotifications),
      async.Go(submitReport),
    )
  ).Await()
}

import "github.com/maniartech/conductor"

func HandleResource(resourceId int) {
  conductor.Async(
    conductor.Sync(
      conductor.Func(keepInfraReady),
      conductor.Func(fetchResource, resourceId),
      conductor.Async(
        conductor.Sync(
          conductor.Func(processA),
          conductor.Func(submitA),
        ).Name("Process A"),
        conductor.Sync(
          conductor.Func(processB),
          conductor.Func(submitB),
        ).Name("ProcessB"),
        conductor.Sync(
          conductor.Func(processC).OnError(handleError).Retry(3, 5*time.Second).Timeout(10*time.Second),
          conductor.Func(submitC).Delay(5*time.Second),
        ).Name("ProcessC"),
      ),
    ).Name("Update Resource"),
    conductor.Async(
      conductor.Func(sendEmails),
      conductor.Func(sendMessages),
    ).Name("Send Notification")
  ).Await()
}
```
