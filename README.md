# Async Await the Go Way  [![Build Status](https://travis-ci.com/maniartech/async.svg?branch=master)](https://travis-ci.com/maniartech/async)

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



* `async.Go(PromiseHandler, ...interfaces{}) *Promsie`

  Executes the function in a new goroutine and returns a promise. The promise can be awaited until the execution is finished and results are returned.

  **Example:**

  ```go
  // Return the pointer to Promise
  promise := async.Go(complexFunction)
  promise.Await() // Awaits here!
  fmt.Printf("Result: %+v", promise.Result)

  // Use await to return the results and error
  result, err := async.Go(complexFunction).Await()
  if err == nil {
    fmt.Printf("Result: %+v", promise.Result)
  }
  ```

* `async.GoC(promises ...*Promise) *Promise`

  Executes the specified promises in the concurrent manner. Returns the promise which can be used to await

* `async.GoQ(promises ...*Promise) *Promise`

  Executes the specified promises in the sequencial manner. Returns the promise which can be used to await

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
```
